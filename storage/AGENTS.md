# CLAUDE.md — storage

This file provides guidance to Claude Code when working with the storage module.

---

## Role

Object storage for all generated static documentation sites. Garage is the target:
S3-compatible, self-hosted, no cloud lock-in. Everything the builder produces lands here.
Everything the reverse proxy serves comes from here.

---

## Path Convention — Hard Contract

All modules that touch storage must follow this layout exactly.

```
Bucket:  <GARAGE_BUCKET>   (single bucket, default: doc-thor-docs)

Paths:
  <project-slug>/<version>/<relative-file-path>

Examples:
  my-api/1.2.0/index.html
  my-api/1.2.0/assets/style.css
  my-api/1.2.0/api-reference/index.html
  my-api/latest/index.html              # see "latest" discussion below
```

---

## Key Decisions to Make

- **Single bucket vs. per-project buckets:** Single bucket with path-based separation is simpler
  to operate. Per-project buckets add isolation and independent ACLs but multiply admin surface.
  **Start with single bucket.** Migrate if isolation becomes a real need.

- **"Latest" version pointer:** Two approaches:
  - Copy: builder also uploads to `<project-slug>/latest/` on every build. Simple, slight storage duplication.
  - Redirect: Nginx rewrites `/latest/` requests to the actual version path. No duplication, slightly more config.
  - **Redirect at Nginx level is preferred.** `server` knows what "latest" is; it tells Nginx. Storage stays dumb.

- **Access control:** Garage supports IAM policies scoped to path prefixes. For v1, all docs are
  public (served via Nginx, no auth at the object level). If per-project auth is needed later,
  add it at the Nginx layer, not the storage layer.

---

## Suggested Stack

- **Garage** — specified, not optional.
- **No wrapper service for v1.** Builder writes via S3 SDK. Reverse proxy reads via Garage HTTP.
- Garage's admin API (port 3903) is available for inspection. Not exposed publicly.
- Initial setup requires running `garage layout assign` + `garage layout apply`,
  then `garage key create` and `garage bucket create` — see `deploy/garage/garage.toml`.

---

## Expected File Structure

```
storage/
├── CLAUDE.md
├── Dockerfile                       # Garage image + init script (layout + bucket + key setup)
└── config/
    ├── .env.example                 # GARAGE_ACCESS_KEY, GARAGE_SECRET_KEY
    └── policy.json.example          # Example IAM policy for builder write access
```

---

## Integration Points

- **Receives uploads from** `builder` — S3 PutObject calls, path = `<slug>/<version>/<file>`.
- **Serves content to** `reverse-proxy` — HTTP GET, Garage's S3 HTTP server.
- **Queried by** `server` — to verify build artifacts exist (ListObjects on a prefix).
