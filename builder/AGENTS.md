# CLAUDE.md — builder

This file provides guidance to Claude Code when working with the builder module.

---

## Role

The build agent. Picks up jobs from `server`, runs the full build pipeline in isolation,
uploads the output to `storage`, and reports back. Multiple builder instances can run in
parallel — this module is designed for horizontal scaling from the start.

The builder is **toolchain-agnostic.** It does not know or care what build tool the user
uses. The user provides a Docker image. The builder starts it, feeds it the repo, and
collects the output. That's the whole job.

---

## Build Pipeline

Every job goes through these stages, in order:

```
1. Pull          — Clone or update the source repository (git)
2. Run           — Start the user's Docker image, mount the cloned repo, wait for exit
3. Collect       — Read the output from the well-known output directory inside the container
4. Upload        — Push output directory to storage (via S3)
5. Report        — POST job result (success/failure + metadata) back to server
```

Each stage is a discrete unit. If stage N fails, the job stops, reports failure, and the
environment is cleaned up. No half-published builds.

---

## The Container Contract

This is the only interface between doc-thor and the user's build logic. Both sides must
agree on it. Nothing else matters.

**What the builder provides to the container:**

- The cloned source repository, mounted read-only at `/repo`.

**What the builder expects from the container:**

- All generated output written to `/output` before the container exits with code 0.
- Exit code 0 = success. Anything else = failure. The builder does not inspect stdout/stderr
  for success signals — it only looks at the exit code.

**That's it.** The container can use any tool, any language, any number of steps internally.
The builder does not care. It does not install plugins. It does not run pre-build scripts.
It does not know what `mkdocs` is. All of that is the image author's responsibility.

### Mount layout inside the container

```
/repo        ← cloned source (read-only mount)
/output      ← generated docs (read-write; builder collects from here after exit)
```

---

## Official Images

doc-thor ships a small set of pre-built images so users don't have to write one from
scratch for common toolchains. These are just regular Docker images that follow the contract
above. Users can use them as-is or as a starting point.

| Image                        | What it does                                          |
|------------------------------|-------------------------------------------------------|
| `doc-thor/builder-mkdocs`    | Runs `mkdocs build` on `/repo`, output to `/output`   |

More images will be added as demand warrants. Users are free to publish and share their
own — they just need to follow the contract.

---

## Isolation Model

Each build runs in its own container. The container is started, it does its work, and it
is removed after the job completes (success or failure). No shared state between builds.
No venvs. No host-level tool management. The image is the isolation boundary.

The builder itself needs Docker access (it runs containers). In Docker Compose that means
mounting the Docker socket or using a dedicated builder runtime (e.g., BuildKit, Podman).
That's an operational concern, not a code concern.

---

## Key Decisions to Make

- **Job queue mechanism:**
  - Poll `server` API on an interval — simple, zero extra infrastructure.
  - Message queue (Redis Streams, NATS, RabbitMQ) — better for scale, adds a service.
  - **Poll for v1.** Swap to a queue when horizontal scaling pressure is real, not hypothetical.

- **Git clone caching:** Cloning large repos on every build is slow. A local cache (bare clone,
  then `git fetch` + worktree) per builder instance speeds things up. Implement after the
  basic pipeline works.

- **Timeout policy:** The container as a whole needs a max duration. Configurable per-project,
  with a global max as a safety net. The builder kills the container if it exceeds the limit.

- **Container runtime:** Docker is the default assumption. Podman is a drop-in if the host
  doesn't want the Docker daemon. Decision is operational, not architectural.

---

## Suggested Stack

- **Go** — same language as `server`. Single static binary. Docker SDK (`github.com/docker/docker/client`)
  is first-class. S3 SDK (`github.com/aws/aws-sdk-go-v2`) is mature. No reason to reach for Python
  now that the builder doesn't need to run MkDocs itself.
- **Docker SDK** — start/stop/wait containers, mount volumes, read exit codes.
- **boto3-equivalent (aws-sdk-go-v2)** — S3 client for storage uploads.
- **net/http** — for reporting back to `server` API.

---

## Expected File Structure

```
builder/
├── CLAUDE.md
├── Dockerfile                       # Builder image: Go binary + Docker socket access
├── agent/
│   ├── main.go                      # Entry point: poll loop or queue consumer
│   ├── pipeline.go                  # Orchestrates stages in order, handles errors
│   ├── stages/
│   │   ├── pull.go                  # git clone / fetch
│   │   ├── run.go                   # Start user container, mount /repo, wait for exit
│   │   ├── collect.go               # Read /output from the container
│   │   └── upload.go                # Uploads output to storage
│   └── report.go                    # POSTs job status to server
└── config/
    └── .env.example                 # SERVER_URL, STORAGE_*, POLL_INTERVAL, CONTAINER_TIMEOUT, etc.
```

---

## Integration Points

- **Receives jobs from** `server` — polls `/api/v1/builds/pending` or consumes from a queue.
  The job payload includes the project's `docker_image` field.
- **Clones repos from** user-configured git sources (GitHub, GitLab, self-hosted Gitea, etc.).
- **Starts containers** using the image specified in the job. Mounts the cloned repo at `/repo`.
  Reads output from `/output` after the container exits.
- **Uploads output to** `storage` — S3 PutObject, path follows the hard contract in `storage/CLAUDE.md`.
- **Reports status to** `server` — POST with job ID, status, duration, error details if any.
