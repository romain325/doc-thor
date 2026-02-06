# AGENTS.md — cli

This file provides guidance to Claude Code when working with the CLI module.

---

## Role

The primary user-facing interface for doc-thor. Every operation that `server` exposes via API
is reachable as a CLI command. No need to open a browser to run this thing.

---

## Command Structure

Every endpoint in `server/api/openapi.yaml` is reachable below. `--json` is a global flag
available on every command; when set, output is raw JSON instead of styled tables.

```
doc-thor project list
doc-thor project create --slug <slug> --name <name> --source-url <url> --docker-image <image>
doc-thor project get    <slug>
doc-thor project update <slug> [--name <name>] [--source-url <url>] [--docker-image <image>]
doc-thor project delete <slug>                  # Confirmation prompt before DELETE

doc-thor build trigger  <slug> [--ref <branch|tag|sha>]
doc-thor build list     <slug> [--limit N] [--offset N]
doc-thor build get      <slug> <build-id>       # Includes logs when available

doc-thor version list       <slug>
doc-thor version publish    <slug> <version>    # PUT {published: true}
doc-thor version unpublish  <slug> <version>    # PUT {published: false}
doc-thor version set-latest <slug> <version>    # PUT {is_latest: true}

doc-thor auth login                             # Interactive: huh prompts for username + password, saves token
doc-thor auth apikey create [--label <label>]   # Prints raw key once; warns it is never shown again
doc-thor auth whoami                            # GET /auth/me — show current user

doc-thor server status                          # GET /health + GET /backends in one view
doc-thor server set    <url>                    # Write target URL into local config

doc-thor integration list                       # List VCS integrations
doc-thor integration create --name <name> --provider <gitlab|github|gitea> --url <url> --token <token> --webhook-secret <secret>
doc-thor integration get    <name>              # Get integration details
doc-thor integration delete <name>              # Delete integration (with confirmation)
doc-thor integration test   <name>              # Test VCS connection

doc-thor discover <integration-name> <scope>    # Discover projects with .doc-thor.project.yaml
doc-thor project import --integration <name> --repo <path> [--register-webhook] [--callback-url <url>]
```

### Endpoint ↔ command map

| CLI command                        | Method | Path                                  |
|------------------------------------|--------|---------------------------------------|
| `server status`                    | GET    | `/health` + `/backends`               |
| `auth login`                       | POST   | `/auth/login`                         |
| `auth apikey create`               | POST   | `/auth/apikey`                        |
| `auth whoami`                      | GET    | `/auth/me`                            |
| `project list`                     | GET    | `/projects`                           |
| `project create`                   | POST   | `/projects`                           |
| `project get`                      | GET    | `/projects/{slug}`                    |
| `project update`                   | PUT    | `/projects/{slug}`                    |
| `project delete`                   | DELETE | `/projects/{slug}`                    |
| `build trigger`                    | POST   | `/projects/{slug}/builds`             |
| `build list`                       | GET    | `/projects/{slug}/builds`             |
| `build get`                        | GET    | `/projects/{slug}/builds/{id}`        |
| `version list`                     | GET    | `/projects/{slug}/versions`           |
| `version publish`                  | PUT    | `/projects/{slug}/versions/{ver}`     |
| `version unpublish`                | PUT    | `/projects/{slug}/versions/{ver}`     |
| `version set-latest`               | PUT    | `/projects/{slug}/versions/{ver}`     |
| `integration list`                 | GET    | `/integrations`                       |
| `integration create`               | POST   | `/integrations`                       |
| `integration get`                  | GET    | `/integrations/{name}`                |
| `integration delete`               | DELETE | `/integrations/{name}`                |
| `integration test`                 | POST   | `/integrations/{name}/test`           |
| `discover`                         | POST   | `/integrations/{name}/discover`       |
| `project import`                   | POST   | `/projects/import`                    |

---

## Local Configuration

Same XDG path as before; format stays TOML. Parsed at startup via `go-toml/v2`.

```toml
# ~/.config/doc-thor/config.toml

[server]
url    = "http://localhost:8000"
api_key = "tok_..."          # session token from login OR an API key
```

---

## Tech Stack (decided)

| Concern                | Choice                                          | Why                                                                 |
|------------------------|-------------------------------------------------|---------------------------------------------------------------------|
| Language               | **Go 1.23**                                     | Single static binary, fast startup, native concurrency for --watch  |
| CLI framework          | **cobra** (`github.com/spf13/cobra`)            | Industry-standard Go CLI; persistent flags, shell completion        |
| Interactive prompts    | **huh** (`github.com/charmbracelet/huh`)        | Typed forms/selects/confirms; used for `auth login` & destructive confirmations |
| Terminal styling       | **lipgloss** (`github.com/charmbracelet/lipgloss`) | Declarative terminal styles; status badges, coloured output         |
| TUI components         | **bubbles** (`github.com/charmbracelet/bubbles`) | Table, list, spinner — used for list commands and --watch polling   |
| TUI runtime            | **bubbletea** (`github.com/charmbracelet/bubbletea`) | MVU loop backing bubbles components and the --watch spinner         |
| HTTP client            | `net/http` (stdlib)                             | CLI is sequential; no external dep needed                           |
| TOML config            | **go-toml/v2** (`github.com/pelletier/go-toml/v2`) | Lightweight; read + write support                                   |

---

## File Structure

```
cli/
├── AGENTS.md
├── go.mod
├── go.sum
├── main.go                              # Entry point — calls cmd.Execute()
├── cmd/
│   ├── root.go                          # Root "doc-thor" command; --json persistent flag; config loader
│   ├── project.go                       # "project" command group (no-op Run, just groups children)
│   ├── project_list.go                  # project list
│   ├── project_create.go                # project create
│   ├── project_get.go                   # project get
│   ├── project_update.go                # project update
│   ├── project_delete.go                # project delete  (huh confirm)
│   ├── build.go                         # "build" command group
│   ├── build_trigger.go                 # build trigger
│   ├── build_list.go                    # build list
│   ├── build_get.go                     # build get  (shows logs)
│   ├── version.go                       # "version" command group
│   ├── version_list.go                  # version list
│   ├── version_publish.go               # version publish
│   ├── version_unpublish.go             # version unpublish
│   ├── version_set_latest.go            # version set-latest
│   ├── auth.go                          # "auth" command group
│   ├── auth_login.go                    # auth login  (huh form)
│   ├── auth_apikey.go                   # auth apikey create
│   ├── auth_whoami.go                   # auth whoami
│   ├── server.go                        # "server" command group
│   ├── server_status.go                 # server status
│   └── server_set.go                    # server set
└── internal/
    ├── client/
    │   └── client.go                    # All HTTP calls. Single place for base URL, Bearer header, error extraction.
    ├── config/
    │   └── config.go                    # Load / Save ~/.config/doc-thor/config.toml
    └── ui/
        ├── output.go                    # Top-level: if --json → marshal; else → hand off to styled renderer
        └── styles.go                    # lipgloss style definitions (badges, table overrides, error/success colours)
```

---

## Output contract

- **Default:** styled. List commands render a `bubbles/table`. Single-resource commands
  render a lipgloss-bordered detail card. Status badges are coloured (green / yellow / red).
  Destructive commands prompt via `huh.NewConfirm()` before acting.
- **`--json`:** every command prints the raw API JSON to stdout, pretty-printed with
  `encoding/json`. No colours, no prompts, no spinner. Piping-friendly.

---

## Error handling

API errors are surfaced as:

```
Error (HTTP 404): project not found
```

No stack traces. Non-zero exit code on failure.

---

## Integration Points

- **Calls** `server` API exclusively via HTTP (`/api/v1/*`). The CLI has no direct
  knowledge of builder, storage, or reverse-proxy.
- **Reads/writes** local config at `~/.config/doc-thor/config.toml`.
