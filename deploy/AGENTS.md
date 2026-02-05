# CLAUDE.md — deploy

This file provides guidance to Claude Code when working with the deploy module.

---

## Role

Docker Compose stack. The single source of truth for how all doc-thor services are wired
together, configured, and deployed. If you want to know how services talk to each other
in a real deployment, start here.

---

## Service Topology

```
docker-compose.yml
├── nginx          ←  reverse-proxy/          (only service exposing ports 80/443)
├── garage         ←  storage/                (internal network only)
├── server         ←  server/                 (internal network only)
├── builder        ←  builder/                (internal network only, scalable via replicas)
└── web            ←  web/                    (served through nginx, placeholder)
```

---

## Network Design

- Single Docker network: `doc-thor-net`.
- **Only Nginx exposes ports externally** (80 and 443). Everything else is internal.
- Garage's admin API (port 3903) is available on the internal network for inspection.
  Can be exposed behind a dedicated Nginx location if needed. Not by default.
- Builder instances have outbound access (they clone git repos). No inbound traffic.

---

## Shared Volumes

| Volume           | Shared Between         | Purpose                                              |
|------------------|------------------------|------------------------------------------------------|
| `nginx-config`   | server ↔ reverse-proxy | Server writes Nginx server blocks; reverse-proxy reads and reloads |
| `garage-data`    | garage only            | Persistent object storage (meta + data). **Do not delete.** |
| `server-data`    | server only            | Persistent database file (SQLite in v1). **Do not delete.** |
| `nginx-certs`    | nginx only             | TLS certificates. Managed by certbot or mounted manually. |

---

## Key Decisions to Make

- **TLS:** Nginx terminates SSL. For production: certbot sidecar with Let's Encrypt.
  For local dev: self-signed cert or plain HTTP on localhost.

- **Secrets:** `.env` file for local development. In production, use Docker secrets or
  an external secrets manager. The `.env` file must never be committed.

- **Builder scaling:** `deploy.replicas` on the builder service. Docker Compose handles
  multiple instances on a single node. For multi-node, migrate to Swarm or Kubernetes —
  but that's a later problem.

- **Persistence:** `garage-data` and `server-data` are the only volumes that matter for
  durability. Back them up. Everything else is ephemeral.

---

## Expected File Structure

```
deploy/
├── CLAUDE.md
├── docker-compose.yml               # Main stack definition (skeleton — see file)
├── docker-compose.override.yml      # Local dev overrides (bind mounts, debug ports, etc.)
├── env/
│   ├── .env.example                 # All env vars with defaults and descriptions
│   ├── nginx.env                    # Nginx-specific env vars
│   ├── garage.env                   # Garage-specific env vars
│   ├── server.env                   # Server-specific env vars
│   └── builder.env                  # Builder-specific env vars
├── garage/
│   └── garage.toml                  # Garage config (single-node dev defaults; see file)
└── nginx/
    └── nginx.conf                   # Base Nginx config (static parts; dynamic parts are generated)
```

---

## Quick Reference (once services are built)

```bash
# Start the full stack
docker compose --env-file env/.env up -d

# Scale builders to 3 instances
docker compose up -d --scale builder=3

# Tail builder logs
docker compose logs -f builder

# Rebuild and restart a single service
docker compose up -d --build server

# Stop everything (data volumes persist)
docker compose down

# Stop everything AND delete data volumes
docker compose down -v
```
