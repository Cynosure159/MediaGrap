# ADR 0001: Foundation stack

- Status: accepted for Phase 0
- Date: 2026-08-31

## Context

MediaGrap needs a low-overhead, self-hosted server that can manage mounted media files and serve a mobile-friendly Web UI from one Docker image.

## Decision

- Use Go 1.26 for the application process.
- Use the standard `net/http` router and `log/slog` instead of a framework for the initial server.
- Use `modernc.org/sqlite`, a pure-Go SQLite driver, to preserve CGO-free cross-platform container builds.
- Use SQLite WAL with embedded ordered SQL migrations.
- Use Vue 3, TypeScript, Vite, and `vite-plugin-pwa` for the frontend.
- Embed the Vite build output into the Go binary at release-build time.
- Use a multi-stage Docker build and distroless non-root runtime image.

## Consequences

- The runtime image contains only the application binary and uses no Java/Node runtime.
- The SQLite driver increases compile time relative to CGO drivers but keeps the container build and cross-architecture path simpler.
- The frontend needs Node only for development and builder stages.
- API/worker boundaries are kept inside the modular monolith so future process separation remains possible.

## Deferred

- Public license selection requires owner approval and is not decided by this ADR.
- Concrete provider contracts, job queue persistence, and source filesystem capability checks belong to later ADRs once their risk spikes are complete.

