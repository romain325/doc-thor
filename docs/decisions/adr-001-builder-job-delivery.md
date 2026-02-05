# ADR-001 — Builder job delivery: polling vs. event-driven

**Status:** Accepted

---

## Context

The builder needs to learn about pending build jobs created on the server. Two broad
approaches were considered:

**Polling.** Each builder instance periodically calls `GET /api/v1/builds/pending`. The
server atomically claims the first pending job (transitions it to `running`) and returns
it. If nothing is pending, it returns 204. Multiple builder replicas poll independently;
the atomic claim on the server side is the only coordination point.

**Server-push (event-driven).** When a build job is created, the server notifies one of
the builder instances directly — via webhook, WebSocket, or SSE. The builder receives the
job and starts working immediately.

A third option, a **message queue** (Redis Streams, NATS, RabbitMQ), was noted as a
future upgrade path but is out of scope for v1.

---

## Decision

**Use polling for v1.**

---

## Rationale

Three constraints in the current architecture make polling the correct default:

1. **Builders have no inbound traffic by design.** The deploy topology gives builders
   outbound access only (they need to clone git repos). A push model inverts this: the
   server would need to discover live builders, maintain a registry, and route jobs to
   them. That turns the server into a scheduler and puts failure-detection burden on it —
   if a builder crashes right after receiving a notification, the server has to detect the
   stale job and re-dispatch.

2. **The latency is irrelevant at this scale.** Build jobs run for seconds to minutes
   (git clone + Docker build + S3 upload). A 5-second poll interval adds latency that no
   user will ever observe. Event-driven delivery is justified when latency matters; it does
   not here.

3. **Polling maps cleanly to horizontal scaling.** Multiple stateless builder replicas
   poll the same endpoint. The server's atomic claim (`SELECT ... FOR UPDATE` or
   equivalent) ensures exactly one builder picks up each job. No leader election, no
   consumer-group coordination, no application-level deduplication. This is a standard
   work-stealing pattern and it works well until the polling rate itself becomes a
   bottleneck on the server — which is a problem for a much larger deployment than v1
   targets.

---

## Consequences

- The server must expose `GET /api/v1/builds/pending` with atomic job claiming. A
  builder that successfully receives a job payload owns it; the server has already
  transitioned it to `running`.
- The server must also expose `POST /api/v1/builds/{id}/result` so that builders can
  report success or failure when the pipeline completes.
- `POLL_INTERVAL` (default 5 s) is the knob to tune if idle polling ever becomes
  noticeable. Raising it costs latency; lowering it costs server load. Neither matters at
  v1 scale.
- When polling becomes a real bottleneck (high job throughput, many builder replicas), the
  upgrade path is a message queue. The server publishes jobs; builders consume. The
  `pending` endpoint disappears. This is a mechanical swap — polling and queue-consumer
  are the same interface from the builder's perspective (`get next job or wait`).
