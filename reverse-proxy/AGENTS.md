# CLAUDE.md — reverse-proxy

This file provides guidance to Claude Code when working with the reverse-proxy module.

---

## Role

Nginx reverse proxy. Single entry point for all doc traffic. Routes requests based on subdomain
to the correct versioned static content in storage. Gets its routing configuration updated
automatically by `server` when projects or versions are published.

---

## Subdomain Convention

```
<project-slug>.docs.<base-domain>            →  latest version of that project
<project-slug>-<version>.docs.<base-domain>  →  specific version
```

Example with `BASE_DOMAIN=docs.example.com` and project `my-api`:

```
my-api.docs.example.com        →  my-api/latest/
my-api-2.1.0.docs.example.com  →  my-api/2.1.0/
```

---

## How It Works

1. `server` detects a new publish event (new project or new version promoted).
2. `server` writes an Nginx `server {}` block to the **shared config volume** (`nginx-config`).
3. Nginx picks up the new config via a reload signal (not a restart — zero downtime).
4. Traffic to the matching subdomain is proxied to the storage backend at the correct path prefix.

---

## Key Decisions to Make

- **Config generation:** Template-based (Jinja2 / Go templates) is preferred. Easier to audit and test than programmatic string building.
- **Static file serving:** Nginx proxies to the storage backend HTTP (Option A) vs. the storage backend exports to a volume Nginx serves directly (Option B). **Option A is preferred** — keeps storage abstracted, no volume coupling.
- **Reload trigger:** `server` writes the file, then calls `nginx -s reload` via a sidecar or a reload-watcher process watching the shared volume. File-watch (inotifywait or equivalent) is simplest.
- **TLS:** Nginx terminates SSL. Certs either mounted manually or managed via a certbot sidecar. Certbot sidecar is cleaner for production.

---

## Suggested Stack

- **Nginx** — specified, not optional.
- **Config generator** — a small script (Python or shell) that reads project metadata from `server` and writes server blocks. Keep it under 100 lines.
- **Reload watcher** — `inotifywait` or a simple polling loop in the same container.

---

## Expected File Structure

```
reverse-proxy/
├── CLAUDE.md
├── Dockerfile                       # Nginx image with config-gen and reload watcher
├── nginx/
│   └── nginx.conf.base              # Static base: logging, defaults, include conf.d/
└── config-gen/
    ├── generate.sh (or .py)         # Reads project list, writes server blocks to conf.d/
    └── templates/
        └── server-block.j2          # Nginx server block template per project/version
```

---

## Integration Points

- **Receives** project/version list from `server` (or reads from a config endpoint).
- **Proxies to** `storage` as origin. Path prefix = `<project-slug>/<version>/`.
- **Exposes** ports 80/443 to the outside world. Nothing else does.
