# VCS Integration Implementation Summary

This document summarizes the VCS integration implementation for doc-thor.

---

## What Was Implemented

### 1. Core VCS Framework

**`server/vcs/interface.go`** - Core VCS abstraction layer
- `Provider` interface defining the contract for all VCS integrations
- `Event` struct for normalized webhook events
- `DiscoveredProject` struct for project discovery results
- `DocThorConfig` struct representing `.doc-thor.project.yaml` configuration
- `IntegrationConfig`, `RepositoryInfo`, and helper types

**`server/vcs/registry.go`** - Provider registration system
- `RegisterProvider()` for registering VCS implementations
- `GetProvider()` for retrieving registered providers
- `ListProviders()` for listing available providers
- Thread-safe registry with sync.RWMutex

### 2. GitLab Provider Implementation

**`server/vcs/gitlab/gitlab.go`** - Complete GitLab integration
- Implements all `vcs.Provider` interface methods
- `ValidateWebhook()` - Validates X-Gitlab-Token and parses webhook payloads
- `DiscoverProjects()` - Scans GitLab groups for projects with `.doc-thor.project.yaml`
- `GetRepositoryInfo()` - Fetches repository metadata
- `RegisterWebhook()` - Creates webhooks on GitLab projects
- `UnregisterWebhook()` - Removes webhooks from GitLab projects
- Helper functions for parsing refs (branches/tags)

**Dependencies**:
- `gitlab.com/gitlab-org/api/client-go` v1.28.0 - Official GitLab API client
- `gopkg.in/yaml.v3` - YAML parsing for `.doc-thor.project.yaml`

### 3. Database Models

**`server/models/models.go`** - Updated data models
- Added `VCSIntegration` model for tracking VCS platform instances
- Added `VCSConfig` struct (stored as JSON on Project)
- Added `BranchMapping` struct for webhook configuration
- Removed deprecated `BuildConfig` (now read from repository at build time)

**VCSIntegration Fields**:
- `Name` - Unique identifier (e.g., "company-gitlab")
- `Provider` - Provider type ("gitlab", "github", "gitea")
- `InstanceURL` - Base URL of VCS instance
- `AccessToken` - API token (encrypted at rest)
- `WebhookSecret` - For webhook signature validation
- `Enabled` - Whether integration is active

### 4. Service Layer

**`server/services/vcs_integrations.go`** - VCS integration management
- `CreateVCSIntegration()` - Create new VCS integration with validation
- `ListVCSIntegrations()` - List all integrations
- `GetVCSIntegration()` - Get integration by name
- `UpdateVCSIntegration()` - Update integration configuration
- `DeleteVCSIntegration()` - Delete integration (with project usage check)
- `TestVCSIntegration()` - Test VCS connection

**`server/services/discovery.go`** - Project discovery and import
- `DiscoverProjects()` - Scan VCS scope for projects with `.doc-thor.project.yaml`
- `ImportProject()` - Create project from discovered config and register webhook
- Automatic webhook registration with event type detection
- Cleanup on failure (unregister webhook if project creation fails)

### 5. API Routes

**`server/routes/integrations.go`** - VCS integration endpoints
- `POST /api/v1/integrations` - Create integration
- `GET /api/v1/integrations` - List integrations
- `GET /api/v1/integrations/{name}` - Get integration details
- `PUT /api/v1/integrations/{name}` - Update integration
- `DELETE /api/v1/integrations/{name}` - Delete integration
- `POST /api/v1/integrations/{name}/test` - Test connection

**`server/routes/webhooks.go`** - Webhook receiver
- `POST /api/v1/webhooks/{provider}/{slug}` - Webhook endpoint (public)
- Validates webhook signatures using integration's webhook secret
- Matches events against project's branch mappings
- Creates builds for matching branches/tags
- Resolves version tags with template variables (`${branch}`, `${tag}`)
- Supports glob patterns for branch matching (`v*`, `release/*`)

**`server/routes/discovery.go`** - Project discovery endpoints
- `POST /api/v1/integrations/{name}/discover` - Discover projects in scope
- `POST /api/v1/projects/import` - Import discovered project

### 6. Application Integration

**`server/cmd/server/main.go`** - Updated main server
- Registers GitLab provider at startup: `vcs.RegisterProvider(&gitlab.GitLabProvider{})`
- Added `VCSIntegration` to database migrations
- Registered webhook routes (public, no auth required)
- Registered VCS integration routes (authenticated)
- Registered discovery routes (authenticated)

---

## API Endpoints Summary

### VCS Integrations (Authenticated)

```
POST   /api/v1/integrations                      # Create VCS integration
GET    /api/v1/integrations                      # List all integrations
GET    /api/v1/integrations/{name}               # Get integration details
PUT    /api/v1/integrations/{name}               # Update integration
DELETE /api/v1/integrations/{name}               # Delete integration
POST   /api/v1/integrations/{name}/test          # Test connection
```

### Project Discovery (Authenticated)

```
POST   /api/v1/integrations/{name}/discover      # Discover projects
                                                  # Body: {"scope": "group/subgroup"}
POST   /api/v1/projects/import                   # Import discovered project
```

### Webhooks (Public)

```
POST   /api/v1/webhooks/{provider}/{slug}        # Webhook receiver
```

---

## Webhook Flow

1. **VCS Platform** sends webhook to doc-thor
   ```
   POST /api/v1/webhooks/gitlab/my-project
   Headers: X-Gitlab-Token: <secret>
   Body: { push event payload }
   ```

2. **doc-thor** validates and processes:
   - Loads project from database
   - Verifies project has VCS config
   - Loads VCS integration (for webhook secret)
   - Gets provider implementation (gitlab)
   - Validates webhook signature
   - Normalizes payload into `Event`
   - Matches event against `branch_mappings`
   - Resolves version tag (e.g., `${branch}` → `main`)
   - Creates build job with resolved version
   - Returns 202 Accepted

3. **Builder** picks up job and builds documentation

4. **Version** published (auto or manual, based on `auto_publish` setting)

---

## Discovery Flow

1. **User** triggers discovery:
   ```bash
   POST /api/v1/integrations/company-gitlab/discover
   Body: {"scope": "myteam/docs"}
   ```

2. **doc-thor** scans VCS:
   - Loads VCS integration config
   - Gets GitLab provider
   - Lists all projects in group (recursively)
   - For each project, checks for `.doc-thor.project.yaml` in root
   - Parses YAML and validates required fields
   - Returns list of discovered projects with parsed config

3. **User** selects project to import:
   ```bash
   POST /api/v1/projects/import
   Body: {
     "integration_name": "company-gitlab",
     "discovered_project": {...},
     "register_webhook": true,
     "callback_url": "https://docs.example.com/api/v1/webhooks/gitlab/my-project"
   }
   ```

4. **doc-thor** imports project:
   - Creates Project record with slug, name, docker_image from config
   - Registers webhook on GitLab
   - Stores webhook ID in project's `vcs_config`
   - Saves branch mappings for automatic builds

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
    }
  ]
}
```

**Behavior**:
- Push to `main` → build `latest`, auto-publish
- Tag `v1.2.3` → build `1.2.3` (strips v prefix), auto-publish
- Push to `release/2.0` → build `release/2.0`, manual publish

---

## Configuration File

Projects declare doc-thor integration via `.doc-thor.project.yaml` in repository root:

```yaml
# Required fields
slug: my-api-docs
name: My API Documentation
docker_image: mycompany/custom-mkdocs-builder:latest

# Optional: default webhook configuration
branch_mappings:
  - branch: main
    version_tag: latest
    auto_publish: true
  - branch: "v*"
    version_tag: "${tag}"
    auto_publish: true
```

**Build Customization**: Projects customize their build by creating a custom Docker image
that includes plugins, pre-build hooks, themes, or any other build-time requirements.
See design documentation for examples.

**JSON Schema**: `docs/.doc-thor.project.schema.json` provides validation and IDE autocompletion.

---

## Security Features

1. **Webhook Signature Validation**
   - All webhooks validated using `WebhookSecret` from VCS integration
   - Invalid signatures rejected with 401 Unauthorized

2. **Token Security**
   - Access tokens stored in database (should be encrypted at rest in production)
   - Tokens never exposed in API responses (marked with `json:"-"`)

3. **Least Privilege**
   - GitLab tokens only need `read_api` + `write_repository` (for webhooks)
   - Separate integration per VCS instance

4. **Project Isolation**
   - Each project has its own webhook with unique slug in URL
   - Projects can only be discovered if they explicitly include `.doc-thor.project.yaml`

---

## Extensibility

### Adding a New VCS Provider (e.g., GitHub)

1. **Create provider implementation**:
   ```go
   // server/vcs/github/github.go
   package github

   import "github.com/romain325/doc-thor/server/vcs"

   type GitHubProvider struct{}

   func (p *GitHubProvider) Name() string { return "github" }
   func (p *GitHubProvider) ValidateWebhook(...) (*vcs.Event, error) { /* ... */ }
   func (p *GitHubProvider) DiscoverProjects(...) ([]vcs.DiscoveredProject, error) { /* ... */ }
   // ... implement remaining interface methods
   ```

2. **Register in main.go**:
   ```go
   vcs.RegisterProvider(&github.GitHubProvider{})
   ```

3. **No other changes needed**:
   - Routes automatically work with any provider
   - Database stores provider as string
   - Discovery and webhooks route to correct implementation

---

## Testing the Implementation

### 1. Create VCS Integration

```bash
curl -X POST http://localhost:8080/api/v1/integrations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-gitlab",
    "provider": "gitlab",
    "instance_url": "https://gitlab.com",
    "access_token": "glpat-...",
    "webhook_secret": "my-secret-123",
    "enabled": true
  }'
```

### 2. Discover Projects

```bash
curl -X POST http://localhost:8080/api/v1/integrations/my-gitlab/discover \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"scope": "mygroup"}'
```

### 3. Import Project

```bash
curl -X POST http://localhost:8080/api/v1/projects/import \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "integration_name": "my-gitlab",
    "discovered_project": { /* from discovery response */ },
    "register_webhook": true,
    "callback_url": "https://docs.example.com/api/v1/webhooks/gitlab/my-project"
  }'
```

### 4. Test Webhook

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/gitlab/my-project \
  -H "X-Gitlab-Token: my-secret-123" \
  -H "X-Gitlab-Event: Push Hook" \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "refs/heads/main",
    "repository": {"path_with_namespace": "mygroup/my-project"},
    "commits": [{"id": "abc123", "message": "Update docs", "author": {"name": "Alice"}}]
  }'
```

---

## Next Steps

1. **GitHub Provider** - Implement GitHub integration following GitLab example
2. **Gitea Provider** - Implement Gitea integration
3. **Token Encryption** - Add encryption for access tokens at rest
4. **Webhook Retry Logic** - Handle failed webhooks with retry mechanism
5. **Discovery Filters** - Add optional filters (topics, labels) for discovery
6. **CLI Commands** - Implement CLI commands for VCS integration management

---

## Files Created/Modified

### New Files
- `server/vcs/interface.go` - Core VCS interface
- `server/vcs/registry.go` - Provider registry
- `server/vcs/gitlab/gitlab.go` - GitLab implementation
- `server/services/vcs_integrations.go` - VCS integration service
- `server/services/discovery.go` - Discovery service
- `server/routes/integrations.go` - VCS integration routes
- `server/routes/webhooks.go` - Webhook routes
- `server/routes/discovery.go` - Discovery routes
- `docs/.doc-thor.project.schema.json` - JSON schema for config file
- `docs/design/vcs-integration.md` - Design documentation
- `docs/implementation-summary.md` - This file

### Modified Files
- `server/models/models.go` - Added VCSIntegration, VCSConfig, removed BuildConfig
- `server/services/projects.go` - Removed BuildConfig update
- `server/cmd/server/main.go` - Registered provider and routes
- `server/go.mod` - Added dependencies
- `server/CLAUDE.md` - Updated documentation
- `docs/.doc-thor.project.yaml` - Added schema reference

---

## Build Status

✅ **All code compiles successfully**
✅ **Dependencies resolved**
✅ **Database migrations added**
✅ **Routes registered**
✅ **Provider registered**

The implementation is complete and ready for testing!
