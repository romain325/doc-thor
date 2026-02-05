# Deployment & Maintenance

Everything you need to get doc-thor running and keep it running. It is not
complicated. It is, however, specific. Skip a step and something breaks in a
way that is not immediately obvious. This guide exists so you don't have to
figure that out yourself.

---

## Prerequisites

- Docker and Docker Compose. Recent versions. Not the ones from 2019.
- A domain name. Or `localhost` if you're just poking at it locally.
- SSH access to the git repositories you want to build. Or HTTPS, if your repos
  are public. The builder needs to clone them.
- The ability to run `docker compose` without sudo errors. Sort that out first.
  Everything downstream assumes it works.

---

## Environment variables

doc-thor is configured entirely through environment variables. No config files
at runtime. No YAML sprawl hiding in a subdirectory somewhere. Every module has
a `.env.example` that documents what it needs. The ones that actually matter:

| Variable | Who needs it | What it is |
|----------|--------------|------------|
| `BASE_DOMAIN` | nginx, server | Your root domain. Subdomains are carved from this. Use `localhost` for local dev. |
| `NGINX_TOKEN` | nginx | Bearer token for config-gen to authenticate with the server. Make it real. |
| `BUILDER_TOKEN` | builder | Bearer token for builders to authenticate with the server. Different from `NGINX_TOKEN`. |
| `STORAGE_ACCESS_KEY` | builder | S3 access key for Garage. Generated during Garage setup. |
| `STORAGE_SECRET_KEY` | builder | S3 secret key. Same story. |
| `STORAGE_BUCKET` | nginx, builder | The Garage bucket name. Default: `doc-thor-docs`. |
| `INITIAL_USER` | server | Username for the first admin account. Created on first startup. |
| `INITIAL_PASSWORD` | server | Password for that account. Change it after you log in. Actually do it. |

The optional ones that are worth knowing about:

| Variable | Default | What it does |
|----------|---------|--------------|
| `HTTP_PORT` | 80 | Host-facing HTTP port. |
| `HTTPS_PORT` | 443 | Host-facing HTTPS port. |
| `NGINX_POLL_INTERVAL` | 10 | Seconds between config-gen polls. Lower = faster routing updates. Higher = less server chatter. |
| `BUILDER_POLL_INTERVAL` | 5 | Seconds between builder job polls. Same trade-off. |
| `BUILDER_REPLICAS` | 1 | Number of builder instances to run. |

---

## First deployment

```bash
# Clone the repo.
git clone <doc-thor-repo>
cd deploy

# Create your .env. Fill in every variable in the table above.
# Don't leave placeholder values. They will bite you silently.
cp env/.env.example env/.env
# ... edit env/.env ...

# Start the stack.
docker compose --env-file env/.env up -d --build

# Verify everything came up.
docker compose ps
```

Wait for all services to show as `running`. If something keeps restarting,
check its logs before doing anything else. The logs will tell you what's wrong.
The service status will not.

---

## Garage first-time setup

This step is easy to miss. Don't miss it.

Garage does not serve buckets as websites by default. You have to opt in.
One command, run once, persists forever:

```bash
docker compose exec garage garage bucket website --allow doc-thor-docs
```

If you skip this, the S3 Web endpoint (port 3902) returns 404 for everything.
Nginx will look healthy. Config-gen will look healthy. The entire system will
appear to be working perfectly — until someone actually tries to open a doc page.
Then: 404. The error message from Garage does not explain why. This page does.

---

## Registering and publishing your first project

```bash
# Register the project.
# --image is the Docker image that runs your build pipeline.
doc-thor project create \
  --name "my-api" \
  --url "https://github.com/you/my-api.git" \
  --image "doc-thor/builder-mkdocs"

# Trigger a build. --tag is the version label that will be published.
doc-thor build trigger my-api --ref main --tag 1.0.0

# Check status. Wait for it to finish.
doc-thor build list my-api

# Once it's successful — promote the version to latest.
# Nothing serves on the bare subdomain until you do this explicitly.
doc-thor version set-latest my-api 1.0.0
```

Your docs are now live at `my-api.<BASE_DOMAIN>`. Version `1.0.0` is also
available at `my-api-1.0.0.<BASE_DOMAIN>`. Future versions work the same way:
build, then promote if you want it to be the default. Promotion is a conscious
decision. "Latest" doesn't move by itself.

---

## Scaling builders

Builds are piling up? Add more builders:

```bash
docker compose up -d --scale builder=3
```

They coordinate automatically. The server hands out jobs atomically: one job,
one builder, no duplicates. No additional configuration. Just change the number.

Scale back down the same way. In-flight builds finish before the container stops,
as long as you're using a normal stop (not `--force-recreate`).

---

## Routine maintenance

### Restarting a service

```bash
docker compose restart <service>
# e.g.: docker compose restart server
```

### Rebuilding after a code change

```bash
docker compose up -d --build <service>
```

### Reading logs

```bash
# Single service:
docker compose logs -f builder

# Everything at once:
docker compose logs -f
```

### Backing up

Two volumes hold persistent state. Everything else is regenerated on startup.
Back these up before upgrades. Back these up periodically. Back these up now
if you haven't already.

```bash
# Garage data — all built documentation.
docker run --rm \
  -v garage-data:/source \
  -v $(pwd):/backup \
  alpine tar czf /backup/garage-data.tar.gz -C /source .

# Server data — project registry, build history, version records.
docker run --rm \
  -v server-data:/source \
  -v $(pwd):/backup \
  alpine tar czf /backup/server-data.tar.gz -C /source .
```

---

## Troubleshooting

### Everything returns 404

The bucket isn't enabled for web serving. This is the single most common
first-time issue, by a comfortable margin.

```bash
docker compose exec garage garage bucket website --allow doc-thor-docs
```

Run it once. It persists. Move on.

### AccessDenied from Garage

Nginx is hitting the wrong port. Port 3900 is the S3 API — it requires
authentication. Port 3902 is the web endpoint — it does not. Check that
`STORAGE_URL` in your environment points to port 3902.

### Builds stuck in "pending"

The builders aren't claiming jobs. Either they're not running, they can't reach
the server, or `BUILDER_TOKEN` is wrong.

```bash
docker compose logs -f builder
```

Look for connection errors, 401s, or panics. One of those three.

### "Permission denied" on git clone

The SSH agent isn't available inside the builder container. `SSH_AUTH_SOCK`
must be set in the shell that runs `docker compose up` — the socket is
bind-mounted, so if the agent isn't running in that shell, there is nothing
to mount.

```bash
eval $(ssh-agent -s)
ssh-add ~/.ssh/id_ed25519   # or whichever key you use
docker compose up -d --build builder
```

### Files download instead of rendering

The files in storage were uploaded without a `Content-Type` header. This
happened before the upload stage was fixed to set it from the file extension.
Trigger a new build for the affected project. Already-uploaded files don't
change retroactively — only new uploads get the correct headers.

### Config-gen keeps logging errors

It's polling the server and getting back something unexpected. Could be a
network issue, a token problem, or the server returning an error. Check both
sets of logs side by side:

```bash
docker compose logs -f nginx
docker compose logs -f server
```

The server logs will have the real error. The nginx logs will have the symptom.
