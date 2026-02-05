# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project: doc-thor

Self-hosted platform for automated technical documentation generation and versioned publishing.
Takes documentation sources (MkDocs-based to start), builds them, stores the static output,
and serves each version under an auto-provisioned subdomain. Everything after the initial
project registration is handled by the pipeline.

---

## Module Map

| Module            | Responsibility                                                                 | Communicates with                        |
|-------------------|--------------------------------------------------------------------------------|------------------------------------------|
| `server/`         | Core API: project registry, build jobs, versions, auth, auto-discovery         | builder, storage, reverse-proxy (config) |
| `builder/`        | Build agents: clone → pre-build → build → upload. Horizontally scalable.       | server (jobs), storage (uploads)         |
| `storage/`        | S3-compatible object storage. Buckets/paths per project+version.               | builder (receives), reverse-proxy (serves)|
| `reverse-proxy/`  | Nginx. Dynamic subdomain routing → versioned docs in storage.                  | storage (origin), server (config push)   |
| `cli/`            | CLI. Every server API operation exposed as a command.                           | server (HTTP client)                     |
| `web/`            | Web UI. Placeholder. Thin client over server API when built.                   | server (HTTP client)                     |
| `deploy/`         | Docker Compose stack. Single source of truth for service wiring.               | all modules                              |
| `docs/`           | Project documentation. Built with doc-thor itself (dogfooding).                | —                                        |

---

## Data Flow

```
  User (CLI or Web)
        │
        ▼
    server (API)
        │
        ├─── triggers build job
        │           │
        │           ▼
        │     builder (agent)
        │           │  clone repo
        │           │  run pre-build hooks
        │           │  run mkdocs build
        │           │  upload output
        │           ▼
        │     storage
        │           ░
        ├─── pushes updated routing config
        │           │
        ▼           ▼
    reverse-proxy (Nginx)
        │
        ▼
    browser → https://<project>[-<version>].docs.<domain>/
```

---

## Shared Conventions

- **Configuration via environment variables.** No scattered config files at runtime.
  `.env` for local dev, real env vars in production. Each module has a `.env.example`.
- **Each module is an independently deployable container.** No shared-filesystem shortcuts between services (except the nginx-config shared volume between server and reverse-proxy, by design).
- **Storage paths are a hard contract:** `<project-slug>/<version>/<file-path>`. All modules that touch storage follow this. No exceptions.
- **Build jobs are idempotent.** Same source commit + same config = same output.
- **Inter-module API contracts are versioned.** The server's API is the contract hub. See `server/CLAUDE.md`.
- **No framework bloat.** Each module picks the lightest tool that does the job. Justify every dependency.

---

## Tech Stack Status

**Not finalized.** Each module's CLAUDE.md lists constraints and preferred candidates.
When a decision is made for a module, update that module's CLAUDE.md first, then update
the `deploy/docker-compose.yml` accordingly.

---

## Supervision Rules (for sub-Claudes)

1. Read the module-level `CLAUDE.md` before touching any module.
2. If you change an inter-module contract (API shape, storage path, config format), update `server/CLAUDE.md` and `deploy/docker-compose.yml`.
3. The `docs/` folder is meant to be built by doc-thor itself. Keep it buildable with MkDocs.
4. Don't add what isn't needed yet. Placeholder is fine. Speculation in code is not.
5. When in doubt about architecture direction, the module CLAUDE.md files are the source of truth.
