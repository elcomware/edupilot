# ADR-010 — Go backend for the Site Server and Cloud

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

The desktop application already embeds the Go application services, domain and ports. Mode B (School Site Server) and Mode C (EduPilot Cloud) must run the *same* business logic, not a reimplementation. That constrains the language of the server before any other consideration.

## Decision

The Site Server and Cloud API are **Go** services that import the same `internal/platform` and `internal/finance` application services and domain as the desktop host. They differ only in the transport adapter and the repository adapters.

```text
React Web
      ↓
HTTPS API
      ↓
Go application          ← same code as the desktop build
      ↓
EduPilot Domain
      ↓
PostgreSQL
```

Deployment: Linux, container, reverse proxy or load balancer, TLS, Go application, PostgreSQL, object storage, backup system.

Cloud is not a separate codebase. It is the same modules behind an HTTP adapter and a PostgreSQL adapter, fed by the outbox and the sync engine from ADR-007.

## Consequences

- Business rules are written once and tested once.
- The desktop edition is a fully functional offline node, not a degraded client.
- Sync must be explicit: posted financial records are immutable, and editable master data carries `version`, `updated_at` and `origin_site` for controlled conflict resolution. Naive last-write-wins is forbidden.
- Go's deployment characteristics (static binaries, small memory, predictable latency) suit both school hardware and containers.

## Alternatives considered

- **A second server language (Node/TypeScript, Python, Java)** — rejected: duplicated business logic, and the two implementations would inevitably diverge.
- **Microservices per domain in the cloud** — rejected: premature; see ADR-002.
- **The desktop app calling the cloud directly for everything** — rejected: schools with poor connectivity must remain fully operational offline.
