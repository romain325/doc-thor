# Technical Reference

The [architecture page](./architecture.md) gives you the map. This page gives you the
territory. If you just want to use the thing, you probably don't need this. If you want
to understand how it actually works — or you're about to modify it — keep reading.

---

## Module internals

### server

**Stack:** Go · Chi · GORM · SQLite

The single source of truth. Every project, build, and version that exists in doc-thor
is registered here. The server does not clone repos. It does not run builds. It does not
serve docs. It decides what should happen and tracks whether it did.

**Data model:**

- **Project** — a slug (the unique handle used everywhere), a source URL, a Docker image
  name, and a creation timestamp. That's it. No configuration sprawl.
- **Build** — belongs to a project. Has a state machine: `pending → running → success | failed`.
  Records the ref, tag, logs, error (if any), and start/finish timestamps. A build is the
  unit of work. Everything the builder does maps to one build record.
- **Version** — belongs to a project. Created when a build succeeds with a non-empty tag.
  Has two flags: `published` (always true on creation) and `is_latest` (false on creation).
  Promotion to latest is an explicit action. See [the "latest" lifecycle](#the-latest-lifecycle).

**Key behaviors:**

- `ClaimPendingBuild` is atomic. It finds the oldest pending build and transitions it to
  `running` in a single transaction. Only one builder gets each job, regardless of how many
  are polling simultaneously. This is a standard work-stealing pattern. The query that scans
  for pending jobs runs with a silent logger — otherwise it logs "record not found" on every
  empty poll cycle, which gets old fast.

- `ReportBuildResult` guards on the build being in `running` state. If a builder tries to
  report a result for a build that isn't running (stale replica, restarted container, creative
  timing), it gets a 409. No double-reporting.

- `ListProjects` returns enriched objects: slug, the list of published version tags, and
  which version is latest. This is the exact shape that config-gen expects. If you change
  this response, config-gen breaks. They are coupled by contract.

---

### builder

**Stack:** Go · Docker SDK · AWS SDK v2

The only module that does actual work. Runs the full build pipeline for each job.
Designed for horizontal scaling from the start: multiple instances, same code, zero
coordination beyond the atomic job claim on the server.

**The pipeline, stage by stage:**

1. **Pull** — Clones the repository at the specified ref. If no ref was given, it clones
   the default branch and resolves the actual branch name via `git rev-parse --abbrev-ref HEAD`.
   The resolved ref is what gets stored. No guessing involved.

2. **Run** — Starts the user's Docker image. Mounts the cloned repo read-only at `/repo`.
   Waits for the container to exit. Exit code 0 = success. Anything else = failure, with
   whatever the container wrote to stdout/stderr as the error log. The builder does not
   inspect output for success signals. It looks at the exit code. That's the contract.

3. **Collect** — Reads everything out of `/output` inside the container after it exits.
   If `/output` is empty or doesn't exist, the build fails. The container either produces
   output or it doesn't. There is no partial credit.

4. **Upload** — Walks the collected files and PutObjects each one into Garage at
   `<slug>/<version>/<relative-path>`. Sets `Content-Type` based on file extension using
   Go's `mime.TypeByExtension`. Unrecognized extensions fall back to `application/octet-stream`.
   Getting this wrong means browsers download files instead of rendering them. This was
   learned the hard way. Every file gets the right header now.

5. **Report** — POSTs the result back to the server. On success with a non-empty tag, the
   server creates a Version record. On failure, the error and logs are stored. The build
   record is the audit trail.

**Docker socket and workspace paths:**

The builder runs user containers via the host Docker daemon — the Docker socket is
bind-mounted into the builder container. This means the build workspace must exist on the
host at the exact same path as inside the builder. `WORKSPACE_DIR` handles this: it is
bind-mounted identically on both sides. If the paths don't match, the user's container
starts but can't see the repo. The build fails with a confusing error about a missing
config file. This is a Docker-in-Docker footgun. It bites once. Now you know it exists.

---

### storage (Garage)

**Two endpoints. Two jobs. Don't mix them up.**

| Port | Name | Role | Who uses it |
|------|------|------|-------------|
| 3900 | S3 API | Authenticated object operations | builder (uploads) |
| 3902 | S3 Web | Unauthenticated static file serving | nginx (serves to browsers) |

The S3 Web endpoint routes by `Host` header. The header must be `<bucket>.web.garage`.
The `root_domain` in `garage.toml` is `.web.garage`. The bucket must be explicitly enabled
for web serving before any of this works:

```bash
garage bucket website --allow <bucket-name>
```

This is a one-time operation. It persists across restarts. If you forget it, port 3902
returns 404 for everything. The error message does not explain why. This page does.

**Storage path contract:**

```
<project-slug>/<version>/<file-path>
```

Builder writes here. Nginx reads from here. Server knows about it. All three agree on
this layout or nothing works. This is not a suggestion. It is the contract.

---

### reverse-proxy

**Stack:** Nginx · Python · Jinja2

Nginx serves traffic. A Python script (`config-gen`) keeps the routing config current.
Both run in the same container. Neither is interesting on its own. Together, they make
subdomains appear and disappear without human intervention.

**The config-gen loop:**

```
loop forever:
    GET /api/v1/projects          (Bearer NGINX_TOKEN)
    for each project:
        render server-block.j2    →  <slug>.conf
        if content changed: write to /etc/nginx/conf.d/
    prune .conf files for projects that no longer exist
    sleep POLL_INTERVAL seconds
```

Transient network errors are logged as warnings. The loop does not crash. It tries again
next cycle. Files prefixed with `00-` are entrypoint-managed and are never touched.

**Subdomain routing:**

Two patterns per project:

- `<slug>.<base-domain>` — the latest version. Shares a `server {}` block with its
  pinned-version counterpart.
- `<slug>-<version>.<base-domain>` — a specific version. Each version gets its own block.

Both proxy to the same place: `<storage-url>/<slug>/<version>/`. The version value differs.
Everything else is identical. The `Host` header on the upstream request is set to
`<bucket>.web.garage` so Garage routes to the correct bucket.

---

### cli

**Stack:** Go · Cobra

A thin HTTP client over the server API. No logic lives here. Every operation is a direct
mapping to an endpoint. If a command does something unexpected, look at the server.
The CLI is just a translator.

---

## The data contracts

These are the shared agreements between modules. Break one and multiple things break
at the same time, in ways that are not immediately obvious.

### Storage path

```
<project-slug>/<version>/<file-path>
```

Builder writes it. Nginx reads it. Server knows about it.

### Build job payload

What the server hands a builder when it claims a job:

```json
{
  "id": "42",
  "project_slug": "my-api",
  "source_url": "https://github.com/you/my-api.git",
  "ref": "main",
  "version": "1.2.0",
  "docker_image": "doc-thor/builder-mkdocs"
}
```

The builder takes this and executes. It does not invent values.

### Project list (for config-gen)

What the server returns to config-gen:

```json
[
  {
    "slug": "my-api",
    "versions": ["1.0.0", "1.1.0", "1.2.0"],
    "latest": "1.2.0"
  }
]
```

Config-gen renders this into Nginx server blocks. Empty `versions` = no server block.
Empty `latest` = only pinned-version subdomains exist.

---

## Authentication

Two tokens. Two audiences. Both required. Neither optional.

| Token | Env var | Used by | Endpoint(s) |
|-------|---------|---------|-------------|
| Nginx token | `NGINX_TOKEN` | config-gen | `GET /api/v1/projects` |
| Builder token | `BUILDER_TOKEN` | builder | `GET /api/v1/builds/pending`, `POST /api/v1/builds/:id/result` |

Both use `Authorization: Bearer <token>`. Missing or invalid = 401.
No silent fallback to unauthenticated access. No "anonymous mode."

---

## The "latest" lifecycle

A version is not automatically latest. This is deliberate.

1. Build succeeds with a tag (e.g., `2.0.0`).
2. Server creates a Version: `published = true`, `is_latest = false`.
3. An operator explicitly promotes it: `version set-latest <project> 2.0.0`.
4. Server marks `2.0.0` as latest, unmarks the previous latest.
5. Next config-gen poll picks up the change. Nginx routes the bare subdomain to `2.0.0`.

If nobody promotes, the bare subdomain keeps pointing at whatever was previously latest.
Publishing a new version does not move traffic. "Latest" is a claim, not an assumption.
