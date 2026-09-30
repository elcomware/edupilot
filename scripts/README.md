# Scripts

Helper scripts for local development. `Taskfile.yml` at the repository root is the canonical
entrypoint; these exist so a developer without Task installed can still work.

| Script | Purpose |
|---|---|
| `dev.ps1` | Run the site server and the frontend dev server together. |
| `check.ps1` | Lint and test the whole repository. |

```powershell
.\scripts\dev.ps1
.\scripts\check.ps1
```

Database migrations are applied by the application at startup and recorded in the schema-version
table. See [`../database/README.md`](../database/README.md).
