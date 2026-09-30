# Database

```text
database/
├── migrations/
│   ├── sqlite/
│   └── postgres/
├── seeds/
│   ├── core/
│   ├── finance/
│   └── demo/
├── factories/
└── fixtures/
```

## Migrations

- **Released migrations are immutable.** Never edit a migration that has shipped. Add a new one.
- One logical change produces one migration per dialect, with the same sequence number so the
  application and the schema stay traceable together.
- Every table carries `organisation_id`, and `campus_id` / `site_id` where appropriate. Composite
  indexes lead with `organisation_id`.
- Money columns are `INTEGER NOT NULL` minor units, always paired with a currency code. Never
  `REAL` and never `FLOAT`.
- Primary keys are `UUIDv7` stored as text or 16-byte blobs, never auto-increment business
  identifiers.
- SQLite migrations must enable foreign keys; the connection layer does this, not the migration.
- Every migration runs inside a transaction and is recorded in a schema-version table.
- Major migrations take an automatic pre-migration backup first.

## Seeds

- `seeds/core/` — reference data every installation needs: chart of accounts skeleton, default
  roles and permissions, numbering sequences, default workflows.
- `seeds/finance/` — finance reference data that depends on configuration: fee categories,
  payment methods, expense classifications.
- `seeds/demo/` — a realistic school used by demos, screenshots and E2E tests only. Never loaded
  into a production organisation.

## Factories and fixtures

Factories generate valid domain objects for tests. Fixtures are checked-in input data — CSV and
XLSX samples for the import pipeline, bank statement samples for reconciliation tests.
