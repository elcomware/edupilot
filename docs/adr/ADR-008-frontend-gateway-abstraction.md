# ADR-008 — Frontend gateway abstraction

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

React components are the first thing developers reach for when they need data. If they call Wails bindings directly, the frontend is welded to the desktop host, and the planned web and cloud clients become a rewrite of the entire UI layer.

## Decision

React components depend on **gateway interfaces** defined in TypeScript, never on a transport.

```text
React Component
      ↓
FinanceGateway (interface)
      ↓
DesktopFinanceGateway → Wails binding     (Mode A)
HttpFinanceGateway   → HTTPS API         (Mode B, Mode C)
```

One adapter is selected at bootstrap, at runtime, in a single place.

Additional rules:

- No SQL, no finance rules and no business invariants in React.
- No component imports a generated Wails binding.
- Frontend code is a client of the application; the backend is the owner of business truth.
- Optimistic UI is permitted only for non-financial interactions; a critical financial operation waits for backend confirmation before showing a posted state.

## Consequences

- The React application can be served to a browser against the Site Server or Cloud API with no business-layer rewrite.
- Transport details, IPC shape and error mapping are isolated in one folder.
- Interfaces must be designed deliberately, which is a small up-front cost.
- A new transport is an implementation, not a migration.

## Alternatives considered

- **Calling Wails bindings from components** — rejected: the fastest path to a prototype and the most expensive to undo.
- **Writing a React API client with axios throughout** — rejected: same coupling, different spelling.
- **Server-side rendering the whole UI** — rejected: incompatible with the desktop-first delivery model.
