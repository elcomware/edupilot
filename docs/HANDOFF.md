# Handoff

Where EduPilot stands, what was verified, and what to do next. Written at the end
of the Academic/People work.

## The one thing to understand first

**Wails v3 binds arguments positionally** — one JavaScript argument per Go
parameter, excluding the context. So a Go method taking a bare `string` has to be
called as `GetPerson("id")`, while its neighbours are called with objects. That
mixed shape is what silently breaks the desktop build: the frontend compiles, the
Go compiles, and the call fails at runtime on the first screen a school opens.

The rule adopted here, enforced by tests: **every bound method takes either
nothing or exactly one request struct, and returns a `Result`.** One object in,
one envelope out, for every operation. Adding a field to a request later never
changes the binding.

Confirm the convention yourself before changing it:

```
cd cmd/desktop && wails3 generate bindings -ts -d <tmpdir>
```

The generated `app.ts` shows the true signatures.

## Verified at handoff

All green on commit:

| Gate | Command |
| --- | --- |
| Format | `gofmt -l ./cmd ./internal` |
| Build | `go build ./...` |
| Vet | `go vet ./...` |
| Tests | `go test ./...` |
| Types | `cd frontend && npx tsc --noEmit` |
| Lint | `cd frontend && npm run lint` |
| Frontend tests | `cd frontend && npm test` (21 passing) |
| Bundle | `cd frontend && npm run build` |

## Architecture rules that still hold

- `cmd` (transport) → `internal/platform/<domain>/application` →
  `internal/platform/<domain>/domain`. Infrastructure implements inward-facing
  ports; it is never called from transport code.
- The desktop tenant is resolved from the installation on every call
  (`App.tenant`). It is **never** read from request data.
- People, roles, relationships and households are organisation-scoped; academic
  facts are year-scoped. A role is `(organisation, person, academic year, role)`.
- Relationships have no start or end dates. Unlink sets `IsActive = false`.
- Wire objects live in `internal/api`. Application code never returns them.
- Empty slices must serialise as `[]`, never `null`, or every list view has to
  guard against null on each render.

## What was just done

- Resolved the Wails binding question above and made the whole transport
  uniform: added `GetPersonRequest`, `GetRelationshipRequest`,
  `UnlinkRelationshipRequest`, `ListTermsRequest`, `ListHouseholdsRequest` and
  `ListPeopleWithRoleRequest`; converted the string-taking methods to take them.
- Added `cmd/desktop/contract_test.go`. It reads the frontend gateway and checks
  it against the Go service, so the two halves cannot drift again:
  every operation resolves to a real method; every bound method takes at most one
  request struct; every bound method is actually called by the gateway; nothing
  wraps a request in `{ request: ... }`; every namespace is either built or
  explicitly listed in `notYetBuilt`; and `VALIDATION_FAILED` stays distinct from
  `NOT_FOUND`.
- Fixed real contract bugs: `platform.campus.App.List` → `ListCampuses`; removed
  the `{ request }` wrappers; `ListStudents` → `ListPeopleWithRole` with the
  Students screen pinning `STUDENT`; the desktop transport no longer sends `{}`
  to methods that take no argument.
- Added the roster operation `ListPeopleWithRole`, with
  `peopledomain.ParseRole` and `api.NewPersonViewWithRole`.
- **Fixed role badges on the real People list.** `ListPeople` used to map people
  with no roles, so badges and role filters only worked in preview. It now loads
  a year's roles for the whole list in one query (`RoleRepository.ListByPersons`,
  `Service.ListRolesForPeople`) and takes an optional `academicYearId`, defaulting
  to the current year. `api.NewPersonViews`, which caused the bug, was deleted so
  it cannot be reused.
- Unexported `App.relationship`; it was bound but nothing called it.

## Next, in order

1. **Academic years screen.** A fresh install currently has no way to open its
   first year, so nothing downstream is usable. There is no UI for
   `academic.listYears`, `getCurrentYear` or `createYear` yet — the gateway calls
   them and nothing renders them. This is the highest-value next step.
2. **Students screen.** `gateway.students.list` now resolves to a real operation,
   but there is still no Students route. Build it on `ListPeopleWithRole` rather
   than by filtering the People page client-side, which cannot see a year it did
   not load.
3. **Preview fixtures** for `ListPeopleWithRole` and academic years, so the new
   screens work under `VITE_PREVIEW=1` like the People screens do.
4. **Tests for untested modules.** `internal/platform/academic` has no test files
   at all, and neither has `people/adapters/postgres`. The SQLite adapters and
   the desktop transport are covered; these are not.
5. **Configurable ID cards.** `docs/adr/ADR-012-configurable-identity-cards.md`
   is accepted and unimplemented. Keep it a separate `platform/identitycards`
   module with versioned templates, permanent issuance history, configurable
   numbering and delegated rendering.

## Gotchas that will bite you

- **This machine runs out of memory.** The Go linker dies with
  `VirtualAlloc ... errno=1455` when free virtual memory drops below roughly
  1.5 GB. Use `GOMAXPROCS=2` and `go test -p 1 ./...`. `AddInProcess.exe` on this
  box holds ~2.4 GB and cannot be terminated; Chrome is the other large consumer.
- **Recreate local databases.** `0001_core_init.sql` changed after the migration
  was released locally, so existing `.db` files fail their checksum. Delete them.
- **`go build ./...` reaches into `node_modules`** and finds
  `frontend/node_modules/flatted/golang/pkg/flatted`, a stray Go file that is not
  part of this project. Harmless today; exclude it if it ever breaks a gate.
- The browser preview needs `VITE_PREVIEW=1 npm run dev` and is dev-only. It must
  never be reachable in a production build.
