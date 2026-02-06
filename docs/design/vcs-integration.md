# VCS Integration Design

Self-hosted GitLab integration (and future GitHub, Gitea, etc.) for automated webhook-triggered builds and project discovery.

---

## Goals

1. **Webhook Integration**: Automatically trigger builds when code is pushed to a registered branch
2. **Project Discovery**: Scan a VCS scope (group/organization) and auto-register projects with documentation
3. **VCS Agnostic**: Clean interfaces that work for any VCS platform (GitLab, GitHub, Gitea, etc.)
4. **Instance Flexibility**: Support self-hosted and cloud-hosted VCS instances

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        doc-thor server                       │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │            VCS Integration Interface                    │ │
│  │  - ValidateWebhook(payload) → Event                     │ │
│  │  - DiscoverProjects(scope) → []DiscoveredProject        │ │
│  │  - GetRepositoryInfo(url) → RepositoryInfo              │ │
│  │  - RegisterWebhook(project, events) → WebhookID         │ │
│  └────────────────────────────────────────────────────────┘ │
│         ▲                    ▲                    ▲          │
│         │                    │                    │          │
│  ┌──────┴──────┐      ┌──────┴──────┐      ┌──────┴──────┐ │
│  │   GitLab    │      │   GitHub    │      │   Gitea     │ │
│  │ Integration │      │ Integration │      │ Integration │ │
│  └─────────────┘      └─────────────┘      └─────────────┘ │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         Webhook Handler (HTTP Endpoint)                 │ │
│  │  POST /api/v1/webhooks/{provider}/{project-slug}        │ │
│  └────────────────────────────────────────────────────────┘ │
│                            │                                  │
│                            ▼                                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         Build Orchestration Service                     │ │
│  │  (existing - creates build jobs)                        │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         ▲                                        │
         │  push event                            │  trigger build
         │                                        ▼
┌────────┴────────┐                     ┌──────────────────┐
│  GitLab/GitHub  │                     │     Builder      │
│    /Gitea       │                     │    (existing)    │
└─────────────────┘                     └──────────────────┘
```

---

## Data Model

### VCSIntegration

New model to track VCS integration configurations per instance.

```go
type VCSIntegration struct {
    Base
    Name         string  `gorm:"uniqueIndex;not null" json:"name"`
    Provider     string  `gorm:"not null" json:"provider"` // "gitlab" | "github" | "gitea"
    InstanceURL  string  `gorm:"not null" json:"instance_url"`
    AccessToken  string  `gorm:"not null" json:"-"` // encrypted at rest
    WebhookSecret string `gorm:"not null" json:"-"` // for webhook signature validation
    Enabled      bool    `gorm:"default:true" json:"enabled"`
}
```

### ProjectVCSConfig

Extends Project with VCS-specific configuration. Stored as JSON on Project model.

```go
// Updated Project model:
type Project struct {
    Base
    Slug        string       `gorm:"uniqueIndex;not null" json:"slug"`
    Name        string       `gorm:"not null" json:"name"`
    SourceURL   string       `gorm:"column:source_url;not null" json:"source_url"`
    DockerImage string       `gorm:"column:docker_image;not null" json:"docker_image"`
    VCSConfig   *VCSConfig   `gorm:"serializer:json" json:"vcs_config,omitempty"`
}

// VCSConfig is serialized as JSON column
type VCSConfig struct {
    IntegrationName string            `json:"integration_name"` // FK to VCSIntegration
    WebhookID       string            `json:"webhook_id,omitempty"` // provider-specific webhook ID
    BranchMappings  []BranchMapping   `json:"branch_mappings"` // which branches build which versions
    AutoRegister    bool              `json:"auto_register"` // discovered via auto-discovery
}

type BranchMapping struct {
    Branch      string `json:"branch"` // e.g., "main", "v*", "release/*"
    VersionTag  string `json:"version_tag"` // e.g., "latest", "${branch}", "${tag}"
    AutoPublish bool   `json:"auto_publish"` // publish immediately after successful build
}
```

**Note**: `BuildConfig` (plugins, pre_build_hooks, mkdocs_overrides) is NOT stored in the database.
It is read from `.doc-thor.project.yaml` in the repository at build time. This keeps configuration
versioned with the code and eliminates database/repository drift.

---

## VCS Integration Interface

Core abstraction implemented by each VCS provider.

```go
package vcs

import (
    "context"
    "net/http"
)

// Provider is the interface all VCS integrations must implement.
type Provider interface {
    // Name returns the provider identifier ("gitlab", "github", "gitea")
    Name() string

    // ValidateWebhook verifies the webhook signature and parses the payload.
    // Returns a normalized Event or an error if invalid/unsupported.
    ValidateWebhook(r *http.Request, secret string) (*Event, error)

    // DiscoverProjects scans the given scope (group, org, namespace) and
    // returns projects that have a .doc-thor.project.yaml file in their root.
    DiscoverProjects(ctx context.Context, config IntegrationConfig, scope string) ([]DiscoveredProject, error)

    // GetRepositoryInfo fetches metadata about a repository (default branch, clone URL, etc.)
    GetRepositoryInfo(ctx context.Context, config IntegrationConfig, repoPath string) (*RepositoryInfo, error)

    // RegisterWebhook creates a webhook on the VCS platform for the given project.
    // Returns the provider-specific webhook ID.
    RegisterWebhook(ctx context.Context, config IntegrationConfig, repoPath string, events []EventType, callbackURL string) (string, error)

    // UnregisterWebhook deletes a webhook by its provider-specific ID.
    UnregisterWebhook(ctx context.Context, config IntegrationConfig, webhookID string) error
}

// IntegrationConfig contains the connection details for a VCS instance.
type IntegrationConfig struct {
    InstanceURL   string
    AccessToken   string
    WebhookSecret string
}

// Event is the normalized webhook event after parsing provider-specific payloads.
type Event struct {
    Type        EventType
    Repository  string // e.g., "mygroup/mydocs"
    Branch      string
    Tag         string
    Commit      string
    CommitMessage string
    Author      string
}

type EventType string

const (
    EventPush   EventType = "push"
    EventTag    EventType = "tag"
    // Future: EventMergeRequest, EventPullRequest, etc.
)

// DiscoveredProject represents a project found during discovery.
type DiscoveredProject struct {
    Name          string
    Path          string // full repo path: "group/subgroup/project"
    CloneURL      string
    DefaultBranch string
    HasDocThor    bool             // true if .doc-thor.project.yaml exists
    DocThorConfig *DocThorConfig   // parsed from .doc-thor.project.yaml
}

// DocThorConfig is parsed from .doc-thor.project.yaml in the repository root.
// This file explicitly declares a project's doc-thor configuration.
type DocThorConfig struct {
    Slug           string            `yaml:"slug"`                  // URL-safe project identifier
    Name           string            `yaml:"name"`                  // Human-readable name
    DockerImage    string            `yaml:"docker_image"`          // Builder image (e.g., "doc-thor/mkdocs-material:latest")
    BuildConfig    BuildConfig       `yaml:"build_config,omitempty"`    // Optional build customization
    BranchMappings []BranchMapping   `yaml:"branch_mappings,omitempty"` // Optional default webhook config
}

// RepositoryInfo is metadata about a single repository.
type RepositoryInfo struct {
    FullPath      string
    DefaultBranch string
    CloneURL      string
    Description   string
}
```

---

## .doc-thor.project.yaml File Format

Projects that want to be discoverable by doc-thor must include a `.doc-thor.project.yaml` file
in their repository root. This file declares the project's doc-thor configuration.

### Minimal Example

```yaml
slug: my-api-docs
name: My API Documentation
docker_image: doc-thor/mkdocs-material:latest
```

### Full Example with Optional Fields

```yaml
# Required: URL-safe project identifier (used in subdomain: my-api-docs.docs.example.com)
slug: my-api-docs

# Required: Human-readable project name
name: My API Documentation

# Required: Docker image for building docs
# Can be a pre-built doc-thor image or a custom image following the builder contract
# Customize the build process by creating your own builder image
# (e.g., add plugins, pre-build hooks, custom themes, code generation, etc.)
docker_image: mycompany/custom-mkdocs-builder:latest

# Optional: Default branch mappings for webhooks
# Can be overridden during import or project configuration
branch_mappings:
  - branch: main
    version_tag: latest
    auto_publish: true
  - branch: "v*"
    version_tag: "${tag}"
    auto_publish: true
  - branch: "release/*"
    version_tag: "${branch}"
    auto_publish: false
```

### Field Reference

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `slug` | Yes | string | URL-safe project identifier. Used in subdomain and API paths. Must be unique. |
| `name` | Yes | string | Human-readable project name displayed in UI. |
| `docker_image` | Yes | string | Docker image used to build the documentation. Must follow builder contract (`/repo` input, `/output` result). Customize the build process by creating your own builder image. |
| `branch_mappings` | No | array | Default webhook configuration. Can be customized during import. |
| `branch_mappings[].branch` | Yes | string | Branch/tag pattern: `main`, `v*`, `release/*`. |
| `branch_mappings[].version_tag` | Yes | string | Target version. Use `${branch}` or `${tag}` for dynamic values. |
| `branch_mappings[].auto_publish` | No | bool | Auto-publish version after successful build. Default: false. |

### Benefits of Explicit Configuration

1. **Intentional Discovery**: Projects opt-in to doc-thor by adding the config file
2. **Self-Describing**: All configuration lives in the repository, versioned with the code
3. **Flexible Builder Images**: Each project can use a different builder image - customize the build process by extending the base image
4. **Centralized Configuration**: Single source of truth for project setup
5. **Import Efficiency**: Discovery can parse full config without additional API calls

### Schema Validation

A JSON Schema is provided at `docs/.doc-thor.project.schema.json` for validation and IDE autocompletion.
Projects can reference it in their YAML file:

```yaml
# yaml-language-server: $schema=https://doc-thor.dev/.doc-thor.project.schema.json

slug: my-project
...
```

---

## GitLab Implementation

```go
package gitlab

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/xanzy/go-gitlab"
    "gopkg.in/yaml.v3"
    "doc-thor/server/vcs"
)

type GitLabProvider struct{}

func (p *GitLabProvider) Name() string {
    return "gitlab"
}

func (p *GitLabProvider) ValidateWebhook(r *http.Request, secret string) (*vcs.Event, error) {
    // Validate X-Gitlab-Token header
    token := r.Header.Get("X-Gitlab-Token")
    if token != secret {
        return nil, fmt.Errorf("invalid webhook token")
    }

    // Parse event type
    eventType := r.Header.Get("X-Gitlab-Event")
    if eventType != "Push Hook" && eventType != "Tag Push Hook" {
        return nil, fmt.Errorf("unsupported event type: %s", eventType)
    }

    // Parse payload
    var payload struct {
        Ref        string `json:"ref"`
        Repository struct {
            PathWithNamespace string `json:"path_with_namespace"`
        } `json:"repository"`
        Commits []struct {
            ID      string `json:"id"`
            Message string `json:"message"`
            Author  struct {
                Name string `json:"name"`
            } `json:"author"`
        } `json:"commits"`
    }

    body, _ := io.ReadAll(r.Body)
    if err := json.Unmarshal(body, &payload); err != nil {
        return nil, err
    }

    event := &vcs.Event{
        Repository: payload.Repository.PathWithNamespace,
    }

    if eventType == "Tag Push Hook" {
        event.Type = vcs.EventTag
        event.Tag = extractTag(payload.Ref) // refs/tags/v1.0.0 -> v1.0.0
    } else {
        event.Type = vcs.EventPush
        event.Branch = extractBranch(payload.Ref) // refs/heads/main -> main
    }

    if len(payload.Commits) > 0 {
        event.Commit = payload.Commits[0].ID
        event.CommitMessage = payload.Commits[0].Message
        event.Author = payload.Commits[0].Author.Name
    }

    return event, nil
}

func (p *GitLabProvider) DiscoverProjects(ctx context.Context, config vcs.IntegrationConfig, scope string) ([]vcs.DiscoveredProject, error) {
    client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
    if err != nil {
        return nil, err
    }

    // List all projects in the group/namespace recursively
    opt := &gitlab.ListGroupProjectsOptions{
        IncludeSubGroups: gitlab.Bool(true),
        ListOptions: gitlab.ListOptions{PerPage: 50},
    }

    var discovered []vcs.DiscoveredProject

    for {
        projects, resp, err := client.Groups.ListGroupProjects(scope, opt, gitlab.WithContext(ctx))
        if err != nil {
            return nil, err
        }

        for _, proj := range projects {
            // Check if project has .doc-thor.project.yaml
            hasDocThor, docThorConfig := p.checkForDocThor(ctx, client, proj)

            if hasDocThor {
                discovered = append(discovered, vcs.DiscoveredProject{
                    Name:          proj.Name,
                    Path:          proj.PathWithNamespace,
                    CloneURL:      proj.HTTPURLToRepo,
                    DefaultBranch: proj.DefaultBranch,
                    HasDocThor:    true,
                    DocThorConfig: docThorConfig,
                })
            }
        }

        if resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }

    return discovered, nil
}

func (p *GitLabProvider) checkForDocThor(ctx context.Context, client *gitlab.Client, proj *gitlab.Project) (bool, *vcs.DocThorConfig) {
    // Check for .doc-thor.project.yaml in root
    file, _, err := client.RepositoryFiles.GetFile(
        proj.ID,
        ".doc-thor.project.yaml",
        &gitlab.GetFileOptions{Ref: &proj.DefaultBranch},
        gitlab.WithContext(ctx),
    )

    if err != nil {
        return false, nil
    }

    // Decode base64 content
    content, err := base64.StdEncoding.DecodeString(file.Content)
    if err != nil {
        return false, nil
    }

    // Parse YAML
    var config vcs.DocThorConfig
    if err := yaml.Unmarshal(content, &config); err != nil {
        return false, nil
    }

    return true, &config
}

func (p *GitLabProvider) GetRepositoryInfo(ctx context.Context, config vcs.IntegrationConfig, repoPath string) (*vcs.RepositoryInfo, error) {
    client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
    if err != nil {
        return nil, err
    }

    proj, _, err := client.Projects.GetProject(repoPath, nil, gitlab.WithContext(ctx))
    if err != nil {
        return nil, err
    }

    return &vcs.RepositoryInfo{
        FullPath:      proj.PathWithNamespace,
        DefaultBranch: proj.DefaultBranch,
        CloneURL:      proj.HTTPURLToRepo,
        Description:   proj.Description,
    }, nil
}

func (p *GitLabProvider) RegisterWebhook(ctx context.Context, config vcs.IntegrationConfig, repoPath string, events []vcs.EventType, callbackURL string) (string, error) {
    client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.InstanceURL))
    if err != nil {
        return "", err
    }

    proj, _, err := client.Projects.GetProject(repoPath, nil, gitlab.WithContext(ctx))
    if err != nil {
        return "", err
    }

    // Map vcs.EventType to GitLab event flags
    pushEvents := false
    tagEvents := false
    for _, e := range events {
        if e == vcs.EventPush {
            pushEvents = true
        }
        if e == vcs.EventTag {
            tagEvents = true
        }
    }

    hook, _, err := client.Projects.AddProjectHook(proj.ID, &gitlab.AddProjectHookOptions{
        URL:                   &callbackURL,
        PushEvents:            &pushEvents,
        TagPushEvents:         &tagEvents,
        Token:                 &config.WebhookSecret,
        EnableSSLVerification: gitlab.Bool(true),
    }, gitlab.WithContext(ctx))

    if err != nil {
        return "", err
    }

    return fmt.Sprintf("%d", hook.ID), nil
}

func (p *GitLabProvider) UnregisterWebhook(ctx context.Context, config vcs.IntegrationConfig, webhookID string) error {
    // Implementation omitted for brevity
    return nil
}

// Helper functions
func extractBranch(ref string) string {
    // refs/heads/main -> main
    const prefix = "refs/heads/"
    if len(ref) > len(prefix) {
        return ref[len(prefix):]
    }
    return ref
}

func extractTag(ref string) string {
    // refs/tags/v1.0.0 -> v1.0.0
    const prefix = "refs/tags/"
    if len(ref) > len(prefix) {
        return ref[len(prefix):]
    }
    return ref
}
```

---

## Server Integration

### New API Endpoints

```
# VCS Integrations
POST   /api/v1/integrations                       # Create VCS integration config
GET    /api/v1/integrations                       # List integrations
GET    /api/v1/integrations/{name}                # Get integration details
PUT    /api/v1/integrations/{name}                # Update integration
DELETE /api/v1/integrations/{name}                # Delete integration
POST   /api/v1/integrations/{name}/test           # Test connection

# Project Discovery
POST   /api/v1/integrations/{name}/discover       # Discover projects in scope
                                                   # Body: {"scope": "group/subgroup"}
                                                   # Response: []DiscoveredProject

POST   /api/v1/projects/import                    # Import discovered project
                                                   # Body: DiscoveredProject + branch mappings

# Webhooks
POST   /api/v1/webhooks/{provider}/{project-slug} # Webhook receiver endpoint
                                                   # Called by VCS platform

POST   /api/v1/projects/{slug}/webhooks/register  # Register webhook on VCS platform
DELETE /api/v1/projects/{slug}/webhooks/unregister # Unregister webhook
```

### Webhook Handler Flow

```go
// routes/webhooks.go
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
    provider := chi.URLParam(r, "provider")
    projectSlug := chi.URLParam(r, "project-slug")

    // 1. Load project from DB
    project := getProject(projectSlug)
    if project.VCSConfig == nil {
        http.Error(w, "project not configured for webhooks", http.StatusBadRequest)
        return
    }

    // 2. Load VCS integration config
    integration := getVCSIntegration(project.VCSConfig.IntegrationName)

    // 3. Get provider implementation
    vcsProvider := vcs.GetProvider(provider) // registry pattern

    // 4. Validate and parse webhook
    event, err := vcsProvider.ValidateWebhook(r, integration.WebhookSecret)
    if err != nil {
        http.Error(w, "invalid webhook", http.StatusUnauthorized)
        return
    }

    // 5. Match event to branch mappings
    for _, mapping := range project.VCSConfig.BranchMappings {
        if matchesBranch(event.Branch, mapping.Branch) || matchesTag(event.Tag, mapping.Branch) {
            // 6. Create build job
            versionTag := resolveVersionTag(mapping.VersionTag, event)
            build := createBuild(project.ID, event.Branch, versionTag)

            // 7. Queue for builder
            queueBuild(build)

            // 8. Auto-publish if configured
            if mapping.AutoPublish {
                onBuildComplete(build.ID, func() {
                    publishVersion(project.ID, versionTag)
                })
            }

            break
        }
    }

    w.WriteHeader(http.StatusAccepted)
}
```

---

## CLI Commands

### VCS Integration Management

```bash
# Add a GitLab instance
doc-thor integration add gitlab \
  --name company-gitlab \
  --url https://gitlab.company.com \
  --token glpat-xxx \
  --webhook-secret <secret>

# List integrations
doc-thor integration list

# Test connection
doc-thor integration test company-gitlab

# Remove integration
doc-thor integration remove company-gitlab
```

### Project Discovery

```bash
# Discover projects in a GitLab group
# Scans for repositories with .doc-thor.project.yaml
doc-thor discover --integration company-gitlab --scope myteam/docs

# Output:
# Found 3 projects with .doc-thor.project.yaml:
#   1. myteam/docs/api-docs (slug: api-docs, image: doc-thor/mkdocs-material:latest)
#   2. myteam/docs/user-guide (slug: user-guide, image: doc-thor/mkdocs:latest)
#   3. myteam/platform/admin-docs (slug: admin-docs, image: custom/sphinx:latest)
#
# Import project? [1-3, all, none]: 1

# Import specific project
doc-thor project import \
  --integration company-gitlab \
  --repo myteam/docs/api-docs \
  --branches main:latest,release/*:${branch} \
  --auto-publish

# This will:
# 1. Create project in doc-thor
# 2. Register webhook on GitLab
# 3. Trigger initial build for 'main' branch
```

### Webhook Management

```bash
# Register webhook for existing project
doc-thor project webhook register my-docs --integration company-gitlab

# Unregister webhook
doc-thor project webhook unregister my-docs

# Show webhook status
doc-thor project show my-docs
# Output includes:
#   VCS Integration: company-gitlab
#   Webhook ID: 12345
#   Branch Mappings:
#     - main → latest (auto-publish)
#     - v* → ${tag} (auto-publish)
```

---

## Branch Mapping Examples

```json
{
  "branch_mappings": [
    {
      "branch": "main",
      "version_tag": "latest",
      "auto_publish": true
    },
    {
      "branch": "v*",
      "version_tag": "${tag}",
      "auto_publish": true
    },
    {
      "branch": "release/*",
      "version_tag": "${branch}",
      "auto_publish": false
    },
    {
      "branch": "dev",
      "version_tag": "dev",
      "auto_publish": false
    }
  ]
}
```

**Behavior:**
- Push to `main` → build version `latest`, auto-publish
- Push tag `v1.2.3` → build version `1.2.3`, auto-publish
- Push to `release/2.0` → build version `release-2.0`, manual publish
- Push to `dev` → build version `dev`, manual publish

---

## Security Considerations

1. **Webhook Secret Validation**: All providers MUST validate webhook signatures using the configured secret
2. **Token Storage**: Access tokens stored encrypted at rest (use `crypto/aes` or dedicated secret manager)
3. **Least Privilege**: VCS tokens should have minimal required scopes:
   - GitLab: `read_api`, `write_repository` (for webhook registration only)
   - GitHub: `repo:status`, `admin:repo_hook`
4. **HTTPS Only**: Webhook callbacks MUST be HTTPS in production
5. **Rate Limiting**: Protect webhook endpoint from abuse (per-project rate limit)

---

## Builder Integration

The builder runs the documentation build inside the specified Docker image.

### Build Process Flow

1. **Server** queues build job with: `project_id`, `ref`, `docker_image`, `source_url`
2. **Builder** picks up job and:
   - Clones repository at specified `ref`
   - Mounts repository at `/repo` (read-only)
   - Runs specified `docker_image` with build command
   - Collects output from `/output` directory
   - Uploads output to storage

### Customizing the Build Process

Projects customize their build by creating a custom Docker image:

```dockerfile
# Example: Custom MkDocs builder with plugins and code generation
FROM doc-thor/mkdocs-material:latest

# Install additional plugins
RUN pip install \
    mkdocs-git-revision-date-localized-plugin \
    mkdocs-macros-plugin \
    mkdocs-swagger-ui-tag

# Install code generation tools
RUN apt-get update && apt-get install -y nodejs npm
RUN npm install -g @stoplight/spectral-cli

# Copy pre-build scripts
COPY scripts/generate-api-reference.sh /scripts/
RUN chmod +x /scripts/*.sh

# Run pre-build hook, then build
ENTRYPOINT ["/bin/bash", "-c", "/scripts/generate-api-reference.sh && mkdocs build"]
```

Then reference it in `.doc-thor.project.yaml`:

```yaml
slug: my-api-docs
name: My API Documentation
docker_image: mycompany/custom-mkdocs-builder:latest
```

This approach gives projects complete control over:
- Build tools and dependencies
- Pre-build code generation
- Custom plugins and themes
- Environment-specific configuration
- Any other build customization

All versioned with the code, no server-side configuration needed.

---

## Extensibility: Adding New Providers

To add support for GitHub, Gitea, Bitbucket, etc.:

1. **Implement `vcs.Provider` interface**:
   ```go
   package github

   type GitHubProvider struct{}

   func (p *GitHubProvider) Name() string { return "github" }
   func (p *GitHubProvider) ValidateWebhook(...) (*vcs.Event, error) { ... }
   func (p *GitHubProvider) DiscoverProjects(...) ([]vcs.DiscoveredProject, error) { ... }
   // ... implement remaining interface methods
   ```

2. **Register provider in server**:
   ```go
   // cmd/server/main.go
   func init() {
       vcs.RegisterProvider(&gitlab.GitLabProvider{})
       vcs.RegisterProvider(&github.GitHubProvider{})
       vcs.RegisterProvider(&gitea.GiteaProvider{})
   }
   ```

3. **No changes needed to**:
   - API endpoints (provider is URL param)
   - Database models (provider stored as string)
   - Build orchestration logic (works with normalized Event)

---

## File Structure

```
server/
├── vcs/
│   ├── interface.go           # Provider interface, Event, DiscoveredProject types
│   ├── registry.go            # Provider registration and lookup
│   ├── gitlab/
│   │   └── gitlab.go          # GitLab implementation
│   ├── github/
│   │   └── github.go          # GitHub implementation (future)
│   └── gitea/
│       └── gitea.go           # Gitea implementation (future)
├── routes/
│   ├── integrations.go        # CRUD for VCSIntegration
│   ├── webhooks.go            # Webhook receiver endpoint
│   └── discovery.go           # Project discovery endpoints
├── services/
│   ├── vcs_integrations.go    # Business logic for VCS config
│   └── discovery.go           # Project discovery and import logic
└── models/
    └── models.go              # Add VCSIntegration, extend Project with VCSConfig
```

---

## Migration Path

### Phase 1: Core Infrastructure
- [ ] Define `vcs.Provider` interface
- [ ] Add `VCSIntegration` model and CRUD API
- [ ] Extend `Project` model with `VCSConfig` JSON field
- [ ] Implement provider registry

### Phase 2: GitLab Implementation
- [ ] Implement `GitLabProvider`
- [ ] Add webhook receiver endpoint
- [ ] Implement branch mapping logic in build orchestration
- [ ] CLI commands for integration management

### Phase 3: Project Discovery
- [ ] Define `.doc-thor.project.yaml` schema and document it
- [ ] Implement `DiscoverProjects` for GitLab (scan for `.doc-thor.project.yaml`)
- [ ] Add YAML parsing and validation
- [ ] Add discovery API endpoint
- [ ] CLI `discover` and `import` commands

### Phase 4: GitHub/Gitea (future)
- [ ] Implement `GitHubProvider`
- [ ] Implement `GiteaProvider`
- [ ] Test provider-agnostic webhook flow

---

## Open Questions

1. **Multi-integration per project?** Should a project support webhooks from multiple VCS integrations? (e.g., mirror on GitLab and GitHub)
   - **Recommendation**: Start with single integration per project. Can extend later if needed.

2. **Webhook delivery retries?** Should doc-thor retry failed builds triggered by webhooks?
   - **Recommendation**: No automatic retries initially. VCS platforms handle webhook retries. Users can manually re-trigger builds.

3. **Discovery filters?** Should discovery support filters beyond `.doc-thor.project.yaml` presence?
   - **Recommendation**: Not initially. Projects opt-in via the config file. Can add topic/label filters later if needed.

4. **Token rotation?** How to handle access token expiration?
   - **Recommendation**: Manual rotation in v1. Add expiry warnings and webhook alerts in v2.

---

## Testing Strategy

1. **Unit Tests**: Mock `vcs.Provider` interface for webhook handler and discovery logic
2. **Integration Tests**: Use GitLab/GitHub test instances or API mocks (go-gitlab test helpers)
3. **End-to-End**: Dockerized GitLab instance + doc-thor stack, trigger real webhook flows
