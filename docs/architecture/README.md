# Architecture

Reference diagrams and structural documentation for EduPilot.

- `module-catalogue.md` — every module, its responsibility and the Core concepts it consumes.
- `data-model.md` — organisation scoping, entity relationships, derivation of balances.
- `event-catalogue.md` — domain events, their payloads and the consumers.

The governing principles live in [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md). The decisions
behind them live in [`../adr/`](../adr/).

## Layer reference

```text
cmd/desktop          Wails host            (Mode A)
cmd/server           Site server           (Mode B)
internal/platform    EduPilot Core
internal/finance     Finance Suite
internal/infrastructure  replaceable adapters
frontend/src         React client
database             migrations and seeds
```

## Module shape

```text
internal/<area>/<module>/
├── domain/         entities, rules, invariants, events
├── application/    commands, queries, transaction boundaries
├── ports/          repository and service interfaces
├── adapters/       sqlite/ postgres/
└── contracts/      transport request and response shapes
```
