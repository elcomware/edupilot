# ADR-006 — Append-only posted financial transactions

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

A bursar will eventually mistype an amount. The instinct is to edit the record. In a school ledger, an edited receipt is unauditable, a corrected invoice may already have been printed and sent to parents, and a re-run payroll is a legal problem. A general-ledger system that permits silent mutation cannot be relied on at audit.

## Decision

Once a financial record is **posted**, it is immutable:

```text
DELETE   ❌
EDIT     ❌
```

Corrections are expressed as new records:

```text
Original transaction
  +
Reversal
  +
Replacement transaction
```

Every reversal references the original transaction it reverses. Student and household balances are **derived from the transaction ledger**, never stored as a mutable `student.balance` column.

Other consequences:

- Credit notes, debit notes, voids and re-posts are first-class, documented operations.
- Each adjustment (discount, waiver, refund) carries reason, authority, approver, supporting document and audit history.
- Document numbering is gap-free per sequence, and cancelled documents still consume their number.

## Consequences

- Financial history is permanently traceable and defensible at audit.
- The schema is append-heavy, which suits SQLite and PostgreSQL equally.
- Reporting reads more rows than a denormalised balance would; balances are maintained as derived read models, never as the source of truth.
- "Undo" is a business workflow with approvals, not a keyboard shortcut.

## Alternatives considered

- **Mutable posted records with an audit table** — rejected: the audit table becomes the real database, and both drift.
- **Full event sourcing** — rejected: reconstructing the entire system from event streams is disproportionate complexity for a school finance product; relational state plus immutable transactions plus events plus audit is sufficient.
- **Soft delete** — rejected: a deleted receipt that still has a number and a printed twin is worse than a reversal.
