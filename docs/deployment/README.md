# Deployment

## Mode A — Standalone desktop (first release)

```text
SCHOOL FINANCE COMPUTER
├── EduPilot.exe              Wails host + Go backend + React frontend
└── EduPilot Data             resolved by the platform storage provider
    ├── database/edupilot.db
    ├── files/                documents and attachments
    ├── backups/
    ├── logs/
    └── runtime/
```

Application files and school data are always separate. The data root is resolved by a platform
storage provider, never hard-coded. `EDUPILOT_DATA_DIR` overrides it for portable installations and
for tests.

Installation: run the Windows installer, choose the data location, create the organisation, create
the first user. No server, no internet connection.

## Mode B — School site server

The server owns the database. Desktop clients on the school LAN talk to the server over HTTP.

```text
                 SCHOOL LAN
                  ┌───────────────┐
                  │ EduPilot Site │
                  │    Server     │
                  └───────┬───────┘
                          │
           ┌──────────────┼──────────────┐
        Cashier        Accountant      Director
```

A SQLite database file is never placed on a network drive for multiple computers to open directly.
For larger deployments the server switches to PostgreSQL behind the same repository ports.

## Mode C — EduPilot Cloud

Linux, container, reverse proxy or load balancer, TLS, Go API, PostgreSQL, object storage, backup
system. The same application services and domain as the desktop build, behind an HTTP adapter and
fed by the durable outbox.

## Backup

```text
Prepare
  → consistent database backup
  → collect files
  → manifest
  → checksums
  → encrypt
  → store
  → verify
```

Major migrations take an automatic pre-migration backup.

## Restore

```text
Select backup
  → validate checksum
  → validate schema version
  → validate organisation
  → create safety backup of current state
  → restore
  → database integrity validation
  → start application
```

## Upgrades

The application version, the database schema version and the sync protocol version are tracked
independently. An upgrade migrates the schema forward only; released migrations are never edited.
