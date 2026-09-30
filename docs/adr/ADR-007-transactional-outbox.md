# ADR-007 — Transactional outbox for integration

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

EduPilot must integrate now (local internal processes: posting to the ledger, generating PDFs, sending notifications) and later with EduPilot Cloud. Publishing events directly to an external system from inside a database transaction is unsafe: the commit can succeed while the publish fails, or the publish can succeed while the commit rolls back.

## Decision

Write a **durable outbox** row in the same transaction as the business change, then publish asynchronously.

```text
outbox_events
├── id
├── organisation_id
├── site_id
├── event_type
├── aggregate_type
├── aggregate_id
├── payload
├── created_at
├── processed_at
└── attempt_count
```

A dispatcher claims unprocessed rows, publishes them, and marks them processed with retry on failure. Inbound cloud traffic lands in an `inbox` table guarded by an idempotency check before any application service runs.

In Mode A the dispatcher serves only local processes, but the table shape is the one that will later feed cloud synchronisation.

## Consequences

- Events and business state cannot diverge.
- The outbox is the natural place to observe "what happened" for debugging and for the sync engine.
- Delivery is at-least-once, so every consumer must be idempotent, keyed on the event ID.
- A backlog is visible and measurable rather than silently lost.

## Alternatives considered

- **Publishing directly from the domain operation** — rejected: dual-write failure.
- **Change data capture on the database** — rejected: couples integration to physical table layout and cannot express business intent.
- **Full event sourcing** — rejected: see ADR-006.
