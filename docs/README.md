# Documentation

```text
docs/
├── architecture/   how the system is put together
├── adr/            decisions and their reasons
├── domains/        per-domain business documentation
├── security/       threat model, permissions, data protection
└── deployment/     installation, upgrades, backup and restore runbooks
```

| Document | Contents |
|---|---|
| [`../ARCHITECTURE.md`](../ARCHITECTURE.md) | Governing principles, layering, invariants, tenancy, events. Read this first. |
| [`adr/`](adr/) | Architecture decision records. One file per decision, immutable once accepted. |
| `architecture/` | Component diagrams, module catalogue, data model, event catalogue, sequence diagrams. |
| `domains/` | Finance, Admissions, Academics and HR documentation. Written by the domain, not by the core. |
| `security/` | Permission model, maker-checker controls, encryption, audit, tenant isolation. |
| `deployment/` | Windows installer, site server, backup and restore procedures, upgrades. |

Documentation follows the code. A module without documentation is not finished, in the same way a
module without tests is not finished.
