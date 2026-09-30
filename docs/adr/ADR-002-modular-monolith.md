# ADR-002 — Modular monolith

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

EduPilot will grow from Finance into Admissions, Academics, HR, Attendance, Library, Transport, Meals, Communication and LMS. The failure mode to avoid is either a single unbounded codebase or premature microservices.

## Decision

Build a **modular monolith** in Go. Code is partitioned into explicit modules under `internal/platform` (EduPilot Core) and `internal/finance` (Finance Suite), each with its own domain, application, ports and adapters.

Modules communicate through **domain interfaces and domain events**, never by reaching into each other's repositories or tables. Shared institutional concepts live in Core and are consumed, not duplicated.

Each module is a valid extraction candidate later, but nothing is deployed as a separate service today.

## Consequences

- One process, one database, one deployment for Mode A and Mode B — appropriate for a school with one finance computer.
- Clear module boundaries are enforced by convention, code review and tests, not by the network.
- Transactional integrity across modules is straightforward (one database transaction).
- The cost is discipline: without enforcement, "modular monolith" decays into a distributed monolith with shared tables.

## Alternatives considered

- **Microservices from day one** — rejected: unjustified operational cost, distributed transactions in a double-entry ledger, and no team size to justify it.
- **Single monolithic package** — rejected: no boundaries, no ownership, unsalvageable as the domain expands.
