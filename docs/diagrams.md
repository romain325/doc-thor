# Diagrams

Visual references for how the system fits together. No diagram will make this
obvious on first glance. These help on second glance.

> **Note:** These diagrams use Mermaid. If they don't render, your MkDocs setup
> needs the `pymdownx.superfences` extension with the mermaid fences configured.
> Don't look at us — that's an MkDocs problem.

---

## Deployment topology

How the services are laid out at runtime. Single Docker network. Only nginx
talks to the outside world. Everything else is internal.

Solid lines are active data transfers. Dotted lines are polling relationships.
The two cylinders at the bottom are the only persistent volumes — the only things
you actually need to back up.

```mermaid
graph TB
    Browser["Browser / CLI"]
    Git["Git Host<br/>GitHub · GitLab · Gitea"]

    subgraph net["doc-thor-net — internal network"]
        Nginx["nginx<br/>+ config-gen<br/>:80 / :443"]
        Server["server<br/>API :8080"]
        Garage["Garage<br/>S3 API :3900<br/>S3 Web :3902"]
        Builder["builder x N<br/>build agent"]
    end

    GD[(garage-data)]
    SD[(server-data)]

    Browser -->|"HTTPS"| Nginx
    Nginx -->|"proxy to S3 Web :3902"| Garage
    Nginx -.->|"polls GET /projects"| Server
    Builder -.->|"polls GET /builds/pending"| Server
    Builder -->|"POST /builds/:id/result"| Server
    Builder -->|"PutObject to S3 API :3900"| Garage
    Builder -->|"git clone"| Git
    Garage -.- GD
    Server -.- SD
```

---

## Build pipeline

What happens from the moment someone triggers a build to the moment docs are live.
The loop at the top is the polling gap — it closes itself every `POLL_INTERVAL` seconds.
The rest is sequential. If any stage fails, the pipeline stops and reports back.

```mermaid
sequenceDiagram
    autonumber
    participant CLI
    participant Server as server
    participant Builder as builder
    participant Git as Git Host
    participant Garage as Garage
    participant Nginx as nginx

    CLI->>Server: POST /builds — trigger build
    Server-->>CLI: 201 Created, status pending

    loop Every POLL_INTERVAL seconds
        Builder->>Server: GET /builds/pending
        alt Job available
            Server-->>Builder: Job payload — atomically claimed
        else Nothing pending
            Server-->>Builder: 204 No Content
        end
    end

    Builder->>Git: git clone at specified ref
    Git-->>Builder: Source repository

    Builder->>Builder: docker run — mount /repo, wait for exit 0
    Builder->>Builder: Collect /output from container

    Builder->>Garage: PutObject per file — S3 API :3900
    Garage-->>Builder: 200 OK

    Builder->>Server: POST /builds/:id/result — success
    Server->>Server: Create Version, published not latest
    Server-->>Builder: 200 OK

    Note over Nginx,Server: Next config-gen poll cycle
    Nginx->>Server: GET /projects
    Server-->>Nginx: Project list with versions
    Nginx->>Nginx: Render server blocks, write if changed
```

---

## Serving a request

What happens when a browser hits a doc subdomain. Three hops. No database query.
No application server in the critical path. Just Nginx reading a routing table
and Garage serving a file.

```mermaid
sequenceDiagram
    participant Browser
    participant Nginx as nginx
    participant Garage as Garage

    Browser->>+Nginx: GET my-api.docs.example.com/
    Note over Nginx: Matches server block<br/>slug = my-api<br/>version = 1.2.0 (latest)
    Nginx->>+Garage: GET /my-api/1.2.0/<br/>Host: doc-thor-docs.web.garage
    Garage-->>-Nginx: index.html — Content-Type text/html
    Nginx-->>-Browser: 200 OK
```
