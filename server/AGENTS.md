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
  created_at    timestamp
  updated_at    timestamp
}
```

When the server queues a build job for the builder, it includes `docker_image` from the
project record. The builder uses it directly — server does not validate or pull the image.
That's the builder's job at runtime.

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
│   └── health.go                    # /api/v1/health, /api/v1/backends
├── models/                          # GORM models (Project, Build, Version, User, APIKey)
├── auth/                            # Auth logic: token validation, API key checks, password hashing
├── services/                        # Business logic: build orchestration, version lifecycle, nginx config gen
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
