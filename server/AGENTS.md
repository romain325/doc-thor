# CLAUDE.md — server

This file provides guidance to Claude Code when working with the server module.

---

## Role

The brain of doc-thor. Project registry, build orchestration, version management,
authentication, and the API that everything else consumes. Both `cli` and `web` are
clients of this API. `builder` polls it for jobs. `reverse-proxy` gets its config from it.

---

## Responsibilities

- **Project registry:** CRUD for documentation projects. Each project has a slug, a git source URL, and a `docker_image` that the builder will use to run the build.
- **Build orchestration:** Create build jobs, queue them, expose their status and logs. The builder pulls from here.
- **Version management:** Track which versions exist per project, which is marked as "latest", publish/unpublish lifecycle.
- **Authentication & authorization:** User accounts, API keys for CLI/automation, session tokens for the web UI. Per-project permission scoping.
- **Auto-discovery:** Health-check and capability-check registered builders and storage backends. Report their status via API.
- **Config push:** Write Nginx routing config when a version is published or unpublished.
- **VCS integration:** Manage connections to VCS platforms (GitLab, GitHub, Gitea). Handle webhook events for automated builds. Discover and auto-register projects from VCS scopes (groups/orgs).

---

## API Contract (planned endpoints)

All endpoints under `/api/v1/`. JSON request/response. API keys via `Authorization: Bearer` header.

```
# Projects
POST   /api/v1/projects                          # Create project
GET    /api/v1/projects                          # List projects
GET    /api/v1/projects/{slug}                   # Get project + config
PUT    /api/v1/projects/{slug}                   # Update project config
DELETE /api/v1/projects/{slug}                   # Delete project (and all versions)

# Builds
POST   /api/v1/projects/{slug}/builds            # Trigger a build (optionally for a specific ref)
GET    /api/v1/projects/{slug}/builds            # List builds (with status, pagination)
GET    /api/v1/projects/{slug}/builds/{id}       # Single build status + logs

# Versions
GET    /api/v1/projects/{slug}/versions          # List published versions
PUT    /api/v1/projects/{slug}/versions/{ver}    # Update version metadata (publish, set latest, unpublish)

# Auth
POST   /api/v1/auth/login                        # Username + password → session token
POST   /api/v1/auth/apikey                       # Generate API key
GET    /api/v1/auth/me                           # Current user info

# System
GET    /api/v1/health                            # Liveness check
GET    /api/v1/backends                          # Discovered builders + storage status

# VCS Integrations
POST   /api/v1/integrations                      # Create VCS integration (GitLab, GitHub, Gitea instance)
GET    /api/v1/integrations                      # List all VCS integrations
GET    /api/v1/integrations/{name}               # Get integration details
PUT    /api/v1/integrations/{name}               # Update integration config
DELETE /api/v1/integrations/{name}               # Delete integration
POST   /api/v1/integrations/{name}/test          # Test VCS connection

# Project Discovery
POST   /api/v1/integrations/{name}/discover      # Discover projects in VCS scope
                                                  # Body: {"scope": "group/subgroup"}
                                                  # Returns: []DiscoveredProject
POST   /api/v1/projects/import                   # Import discovered project with VCS config
                                                  # Body: DiscoveredProject + branch mappings

# Webhooks
POST   /api/v1/webhooks/{provider}/{slug}        # Webhook receiver (called by VCS platform)
                                                  # Validates signature, creates build job
POST   /api/v1/projects/{slug}/webhooks/register # Register webhook on VCS platform
DELETE /api/v1/projects/{slug}/webhooks/unregister # Unregister webhook from VCS platform
```

---

## Project Model

The Project is the central entity. Every other resource (build, version) belongs to one.

```
Project {
  id            uint
  slug          string   # URL-safe unique identifier. Immutable after creation.
  name          string   # Human-readable name.
  source_url    string   # Git repository URL. Builder clones from here.
  docker_image  string   # Docker image the builder will run. Must follow the builder contract
                         # (/repo mounted read-only, output written to /output).
  vcs_config    json     # Optional. VCSConfig: integration name, webhook ID, branch mappings.
  created_at    timestamp
  updated_at    timestamp
}

VCSConfig {
  integration_name  string           # FK to VCSIntegration
  webhook_id        string           # Provider-specific webhook ID
  branch_mappings   []BranchMapping  # Which branches/tags trigger builds
  auto_register     bool             # True if discovered via auto-discovery
}

BranchMapping {
  branch        string  # Pattern: "main", "v*", "release/*"
  version_tag   string  # Target version: "latest", "${branch}", "${tag}"
  auto_publish  bool    # Publish immediately after successful build
}
```

**Build Configuration**: Build-specific settings (plugins, pre-build hooks, mkdocs overrides) are
NOT stored in the database. They are read from `.doc-thor.project.yaml` in the repository at build
time. This keeps configuration versioned with the code.

When the server queues a build job for the builder, it includes `docker_image` from the
project record. The builder uses it directly — server does not validate or pull the image.
That's the builder's job at runtime. The builder will also read `.doc-thor.project.yaml` from
the cloned repository to get build-specific configuration.

When a project has `vcs_config`, webhook events from the VCS platform automatically trigger
builds for matching branches/tags according to the branch mappings.

---

## VCS Integration Architecture

The VCS integration layer enables automatic builds via webhooks and project auto-discovery
from VCS platforms (GitLab, GitHub, Gitea). It's designed to be platform-agnostic with
a clean provider interface.

### Provider Interface

All VCS integrations implement the `vcs.Provider` interface:

```go
type Provider interface {
    Name() string
    ValidateWebhook(r *http.Request, secret string) (*Event, error)
    DiscoverProjects(ctx context.Context, config IntegrationConfig, scope string) ([]DiscoveredProject, error)
    GetRepositoryInfo(ctx context.Context, config IntegrationConfig, repoPath string) (*RepositoryInfo, error)
    RegisterWebhook(ctx context.Context, config IntegrationConfig, repoPath string, events []EventType, callbackURL string) (string, error)
    UnregisterWebhook(ctx context.Context, config IntegrationConfig, webhookID string) error
}
```

### Webhook Flow

1. VCS platform (GitLab/GitHub) sends webhook to `POST /api/v1/webhooks/{provider}/{slug}`
2. Handler loads project's `vcs_config` and the associated `VCSIntegration` config
3. Provider validates webhook signature and normalizes payload into `Event{type, branch, tag, commit}`
4. Handler matches event against project's `branch_mappings` (e.g., `main → latest`)
5. If match found, create build job with resolved version tag
6. If `auto_publish` enabled for that mapping, publish version after successful build

### Project Discovery Flow

1. User/CLI calls `POST /api/v1/integrations/{name}/discover` with `scope` (e.g., `myteam/docs`)
2. Discovery service uses provider's `DiscoverProjects()` to scan VCS scope
3. Provider checks each project for `.doc-thor.project.yaml` in repository root
4. If found, parses file to extract slug, docker_image, branch_mappings, and other config
5. Returns list of `DiscoveredProject` with parsed config from `.doc-thor.project.yaml`
6. User selects projects to import via `POST /api/v1/projects/import`
7. Import creates Project using config from file, registers webhook on VCS platform, stores webhook ID in `vcs_config`

**Note**: Projects must include a `.doc-thor.project.yaml` file to be discoverable. This explicit opt-in
ensures only intended projects are registered and gives each project control over its configuration.

### VCS Integration Model

```go
type VCSIntegration struct {
    Name          string  // Unique identifier, e.g., "company-gitlab"
    Provider      string  // "gitlab" | "github" | "gitea"
    InstanceURL   string  // Base URL of VCS instance
    AccessToken   string  // Encrypted API token
    WebhookSecret string  // For webhook signature validation
    Enabled       bool
}
```

Stored per-instance. Multiple integrations can coexist (e.g., `company-gitlab`, `github-cloud`).

### Extensibility

Adding a new VCS provider (e.g., Bitbucket):

1. Implement `vcs.Provider` interface in `server/vcs/bitbucket/bitbucket.go`
2. Register provider in `cmd/server/main.go`: `vcs.RegisterProvider(&bitbucket.BitbucketProvider{})`
3. No changes needed to API routes, database models, or build orchestration

See `docs/design/vcs-integration.md` for full design details, sequence diagrams, and examples.

---

## Key Decisions to Make

- **Database:** SQLite for v1 — zero config, single file, perfectly fine for single-node.
  Swap to PostgreSQL when multi-node or concurrent-write pressure requires it.
  Use an ORM that abstracts the backend so the swap is mechanical.

- **Auth:** Username + password + API keys for v1. No OAuth/OIDC yet — add it when someone
  actually needs it, not before.

- **Build job state machine:** Jobs go through `pending → running → success | failed`.
  Server owns this state. Builder reports transitions. If a builder dies mid-build,
  server detects the stale job (timeout) and marks it failed.

- **Nginx config push:** Server writes server-block files to a shared volume.
  The reverse-proxy module watches that volume for changes and reloads Nginx.
  This is the only inter-module shared volume, and it's intentional.

---

## Suggested Stack

- **Go + Chi** (`go-chi/chi`) — minimal idiomatic router, zero code-gen, single static binary. Concurrency via goroutines is a natural fit for build-job orchestration.
- **GORM** — ORM that supports SQLite (`modernc.org/sqlite`, pure-Go, no CGO) and PostgreSQL with the same code. Swap is mechanical.
- **`net/http` + `encoding/json`** — stdlib handles serialization and validation at this scope. No extra framework needed.
- **VCS Client Libraries:**
  - GitLab: `github.com/xanzy/go-gitlab` — comprehensive, actively maintained
  - GitHub: `github.com/google/go-github/v57/github` — official SDK
  - Gitea: `code.gitea.io/sdk/gitea` — official SDK

---

## Expected File Structure

```
server/
├── CLAUDE.md
├── Dockerfile                       # Multi-stage: build static binary → scratch/distro-less image
├── go.mod
├── go.sum
├── cmd/
│   └── server/
│       └── main.go                  # Entry point: config loading, DB init, router wiring, http.ListenAndServe
├── routes/
│   ├── projects.go                  # /api/v1/projects
│   ├── builds.go                    # /api/v1/projects/{slug}/builds
│   ├── versions.go                  # /api/v1/projects/{slug}/versions
│   ├── auth.go                      # /api/v1/auth
│   ├── health.go                    # /api/v1/health, /api/v1/backends
│   ├── integrations.go              # /api/v1/integrations (VCS config CRUD)
│   ├── webhooks.go                  # /api/v1/webhooks/{provider}/{slug} (webhook receiver)
│   └── discovery.go                 # /api/v1/integrations/{name}/discover, /api/v1/projects/import
├── models/                          # GORM models (Project, Build, Version, User, Token, VCSIntegration)
├── auth/                            # Auth logic: token validation, API key checks, password hashing
├── services/                        # Business logic: build orchestration, version lifecycle, nginx config gen
│   ├── vcs_integrations.go          # VCS integration management
│   └── discovery.go                 # Project discovery and import logic
├── vcs/                             # VCS integration layer
│   ├── interface.go                 # Provider interface, Event, DiscoveredProject types
│   ├── registry.go                  # Provider registration and lookup
│   ├── gitlab/
│   │   └── gitlab.go                # GitLab Provider implementation
│   ├── github/
│   │   └── github.go                # GitHub Provider implementation (future)
│   └── gitea/
│       └── gitea.go                 # Gitea Provider implementation (future)
├── discovery/                       # Builder and storage health checks
└── config/
    └── .env.example                 # DATABASE_URL, STORAGE_*, NGINX_CONFIG_DIR, etc.
```

---

## Integration Points

- **Exposes API to** `cli` and `web` — the only public-facing service (behind Nginx).
- **Queues jobs for** `builder` — builder polls `/api/v1/builds/pending` (or equivalent).
- **Writes config for** `reverse-proxy` — server-block files on the shared nginx-config volume.
- **Verifies artifacts in** `storage` — ListObjects to confirm a build actually produced output.
- **Receives webhooks from** VCS platforms (GitLab, GitHub, Gitea) — validates signatures, triggers builds.
- **Queries VCS APIs for** project discovery, repository metadata, webhook registration.
