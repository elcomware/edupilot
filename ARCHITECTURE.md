# EduPilot Architecture

> **EduPilot Core owns institutional concepts. Each business domain owns its own business rules. Interfaces own communication between domains. Infrastructure is replaceable. User interfaces are clients of the application, never owners of business truth.**

> **Finance is EduPilot's first commercial domain—not EduPilot's architecture.**

## 1. The path of a request

```text
React Component
      ↓
Frontend Gateway (interface)
      ↓
Wails Transport  (today)      HTTP Transport (later)
      ↓
Application Services
      ↓
Domain
      ↓
Repository Interfaces
      ↓
SQLite Adapter (today)       PostgreSQL Adapter (later)
```

Wails is one delivery adapter, not the architecture. The same application services, domain and ports serve the desktop, the site server and the cloud.

## 2. Dependency rule

```text
React → Transport → Application → Domain
Infrastructure → implements Domain/Application ports
```

Forbidden:

```text
Domain → React        ❌
Domain → Wails        ❌
Domain → SQLite       ❌
Domain → PostgreSQL   ❌
Domain → HTTP         ❌
```

## 3. Core versus domains

**EduPilot Core** builds these once and every domain consumes them:

```text
Organisation · Campus · Academic Year · Academic Structure · People · Students
Households · Employees · Users · Roles · Permissions · Workflows · Approvals
Audit · Documents · Numbering · Notifications · Settings · Import/Export
Files · Search · Backups
```

**Finance must not own these.** A student belongs to Core. Finance owns the student's *financial sub-ledger*, and nothing else. Academics and HR consume the same Person and Student records rather than creating parallel tables.

## 4. The module pattern

Every module — Core or domain — follows the same predictable shape:

```text
internal/<area>/<module>/
├── domain/         entities, value objects, rules, invariants, domain events
├── application/    commands, queries, transaction boundaries
├── ports/          repository and service interfaces
├── adapters/
│   ├── sqlite/     SQLite implementation of the ports
│   └── postgres/   PostgreSQL implementation of the ports
└── contracts/      transport-facing request/response shapes
```

A new module ships: domain, application commands, application queries, permissions, repository port, database migration, events, audit rules, frontend feature, transport contract, tests, documentation.

## 5. Financial integrity

These invariants are enforced by automated tests, and a production release fails if any of them fail:

```text
Debits = Credits                              A closed accounting period receives no postings
Posted journals cannot be edited              Posted payroll runs are immutable
Invoice and receipt numbers are unique       Every reversal references its original
Allocation ≤ payment value                   Allocation ≤ permitted invoice balance
Stock quantity derives from movements        Every posted transaction identifies its actor
Every sensitive change produces an audit record
```

Student balances are **derived from transactions**, never stored as a mutable field. Corrections use original + reversal + replacement, never deletion.

## 6. Tenancy, money, identity

- **Tenancy:** every relevant record carries `organisation_id`, and where appropriate `campus_id` and `site_id`. Boundaries are enforced in repositories, services, permissions and tests — never only in the frontend.
- **Money:** `amount_minor int64` + `currency`. No floating point, anywhere. Exchange rates use exact decimal/rational representation.
- **Identity:** UUIDv7 everywhere, so School A, School B, an offline Node C and the cloud can all create objects independently and still converge.

## 7. Events and integration

```text
Local transaction → SQLite commit → outbox_events → (sync worker) → Cloud API
Cloud → sync → inbox → idempotency check → application service → local database
```

The durable outbox powers local internal processes first, then becomes the foundation of cloud synchronisation. Finance must not use naïve last-write-wins: posted financial records are immutable, and editable master data carries `version`, `updated_at` and `origin_site` for controlled conflict resolution.

This is **not** full event sourcing. EduPilot is a relational database with immutable financial transactions, domain events and an audit log.

## 8. Frontend

React components never import Wails bindings and never contain SQL or finance logic. They call a gateway interface; a desktop implementation and an HTTP implementation satisfy it. User-facing text lives in i18n keys — English and French from day one — and academic terminology (Grade / Class / Year / Niveau / Classe / Form) is configurable rather than hard-coded.

## 9. Deployment

- **Mode A — Standalone desktop:** EduPilot.exe (Wails + Go + React) with EduPilot Data stored separately: `database/edupilot.db`, `files/`, `backups/`, `logs/`, `runtime/`. The OS-specific path is resolved by a platform storage provider.
- **Mode B — Site server:** the server owns the database. Desktop clients talk to the server. A SQLite file is never opened directly from multiple machines on a network drive.
- **Mode C — Cloud:** Go API + PostgreSQL + object storage behind TLS.

An EduPilot Node is simply "the thing that owns a school's local operational state" — the desktop app for a small school, the site server for a large one. That is the migration path.

## 10. What we explicitly do not do

```text
❌ Finance logic inside React        ❌ Floating-point money
❌ SQL inside React                  ❌ Editing posted journals
❌ Finance owning students           ❌ Deleting receipts
❌ Academics owning students         ❌ Hard-coded fee categories
❌ Separate person records per module ❌ Hard-coded country tax logic in core payroll
❌ Module-specific user systems       ❌ Hard-coded Grade 1 / Form 1 / CP structures
❌ Module-specific audit/file/workflow systems
❌ Generic "utils" dumping ground     ❌ Premature microservices
```

## 11. Versioning

Three independent versions, never one number:

```text
EduPilot Application   v1.4.0
Database Schema        v37
Sync Protocol          v3
```

## 12. Decisions

Major decisions live in [`docs/adr/`](adr/). Released migrations are immutable — add new ones.
