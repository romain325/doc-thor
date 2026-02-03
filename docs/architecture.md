# Architecture

doc-thor is split into focused, independently deployable modules. No single module knows
more about the system than it needs to.

---

## Modules

### server

The API. The registry. The place where all decisions about what exists and what should be
built are made. Everything else either asks it questions or does what it says.

### builder

The workhorse. Takes a build job, runs the full pipeline (clone → pre-build → build →
upload), and reports back. Designed to scale horizontally: throw more builders at it
if builds are slow.

### storage

Holds the output. Generated static sites live here. S3-compatible object storage, self-hosted.
Simple, no opinions, just bytes on disk.

### reverse-proxy

Nginx. Routes subdomains to the right version of the right project in storage. Gets its
routing config updated automatically when new projects or versions are published.
You don't touch it after setup.

### cli

How you actually interact with the system day-to-day. Every API operation, as a command.

### web

The UI. Exists as a placeholder. Will eventually be a dashboard for managing projects,
versions, and users. Not currently a priority.

---

## How it fits together

The server orchestrates. The builder executes. Storage stores. Nginx serves.
The CLI talks to the server. That's it.

---

## Deployment

Everything runs in Docker. A single `docker-compose.yml` in `deploy/` wires it all up.
One command to start the entire stack. No Kubernetes required unless you actually need it.
