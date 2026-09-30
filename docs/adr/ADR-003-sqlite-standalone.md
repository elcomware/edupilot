# ADR-003 — SQLite for standalone deployments

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

The first edition targets one school finance computer, often on an unreliable internet connection. The database must be zero-admin, file-based, fully backed up by copying a file, and must not require a database administrator.

## Decision

Use **SQLite** as the standalone (Mode A) and small site-server database, accessed through the repository port implementations in `internal/infrastructure/database`. The database file lives in the EduPilot data root, separate from the application files.

Use **PostgreSQL** for larger site-server and all cloud deployments, behind the same repository ports.

A SQLite file is **never** placed on a network drive for multiple computers to open directly. Multi-user deployments talk to the EduPilot Site Server, which owns the file exclusively.

## Consequences

- Backup is a file copy plus a manifest and checksums; restore is validated before it replaces live data.
- School IT staff can copy the data folder to an external drive.
- SQLite constraints shape the schema: enable foreign keys explicitly, use WAL, and keep long-running write transactions short.
- The adapter layer must avoid vendor-specific SQL so the PostgreSQL adapter stays viable.

## Alternatives considered

- **PostgreSQL everywhere** — rejected: an installation and upgrade burden on school hardware for no benefit at standalone scale.
- **Opening SQLite directly from multiple LAN clients** — rejected: file locking over SMB/NFS is unsafe and will corrupt data.
