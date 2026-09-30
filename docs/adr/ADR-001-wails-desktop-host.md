# ADR-001 — Wails 3 as the desktop host

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

EduPilot's first commercial edition is a Windows desktop application for a school's finance office. It must install cleanly, run offline, feel native, and be operable on a modest school computer without a separate server or a container runtime.

## Decision

Use **Wails 3** as the desktop host. The React frontend renders inside the Windows system WebView2, the Go application services are compiled into the same `EduPilot.exe`, and Wails provides window management, menus, dialogs, file pickers, printing hooks and OS integration.

Wails is a **delivery adapter only**. It is not the architecture. Its bindings are never imported by React components directly and never referenced from domain code.

## Consequences

- One installer, one process, one executable. No runtime prerequisite beyond Windows.
- Frontend and backend ship together; no version skew between client and server.
- The Windows system WebView2 runtime is present on supported Windows versions, so there is no bundled browser engine.
- The Go toolchain and WebView2 headers are required to build.
- A second adapter (HTTP) is required later for the Site Server and Cloud modes, which is exactly why the gateway abstraction exists in the frontend.

## Alternatives considered

- **Electron + Node backend** — rejected: larger memory footprint, a second language runtime in the box, and slower cold start on school hardware.
- **Tauri (Rust host)** — rejected: introduces a Rust toolchain alongside Go for no benefit here, and splits backend language across the codebase.
- **Go + server-rendered web UI** — rejected: the finance UI is a dense, interactive, table-heavy application; a component model is the right fit.
