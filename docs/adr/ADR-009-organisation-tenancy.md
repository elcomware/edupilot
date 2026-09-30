# ADR-009 — Organisation-scoped multi-tenancy from day one

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

A single EduPilot installation may serve a nursery, a primary school, a K–12 school, a multi-campus school or a school group. Retrofitting tenancy into a schema that assumed one school means rewriting every query, every index and every report — after real data exists.

## Decision

Tenancy is structural from the first migration.

- Every relevant record carries `organisation_id`, and where appropriate `campus_id` and `site_id`.
- The tenant scope is part of every repository method signature, not an optional filter.
- Organisation boundaries are enforced in repositories, application services, permission checks and tests.
- Composite indexes lead with `organisation_id`.
- The frontend may filter for usability, but is never the enforcement point.

## Consequences

- Multi-campus and school-group support arrives without a migration project.
- Queries are marginally more verbose and indexes slightly larger — an acceptable cost for correctness.
- Cross-organisation reporting is an explicit, permission-gated capability rather than an accident.
- Import and restore flows must validate organisation identity before writing.

## Alternatives considered

- **One database per school** — rejected: an operational and licensing burden at installation scale, and it complicates any future group consolidation; a single file with strict scoping is right for Mode A.
- **Add `organisation_id` when the second school signs up** — rejected: retrofitting tenancy into live financial data is one of the most expensive refactors in the product.
- **Rely on the frontend to filter** — rejected: trivially bypassed, and a data-isolation defect in a finance system is not survivable.
