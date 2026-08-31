# MediaGrap repository instructions

## Scope

MediaGrap is a lightweight, self-hosted media metadata manager. The target production shape is a single Docker container containing a Go server and an embedded Vue PWA.

## Required workflow

- Read `README.md`, `README-zh.md`, and the relevant design documents before changing architecture or behavior.
- Prefix terminal commands with `rtk`, as required by `/Users/cy/.codex/RTK.md`.
- Keep detailed human-facing project documentation under `docs/`. Root `README.md`, `README-zh.md`, and `AGENTS.md` are the only exceptions.
- Preserve unrelated user changes and never rewrite media files without an explicit, previewable operation.
- Prefer small, reviewable commits and conventional commit messages.

## Architecture constraints

- Backend: Go, organized as a modular monolith with explicit domain and adapter boundaries.
- Frontend: Vue 3, TypeScript, Composition API, responsive and installable as a PWA.
- Default database: SQLite in WAL mode. Keep repository interfaces portable enough for a future PostgreSQL adapter.
- Production: one non-root container and one application process; embed compiled web assets in the Go binary.
- Filesystem writes must use plan/preview, path validation, conflict detection, temporary-file-plus-atomic-rename where supported, and an audit record.
- Scraper integrations must sit behind provider interfaces. Do not leak provider payloads into domain models.
- Long-running work must use persisted jobs with cancellation, retry policy, progress, and structured logs.
- Never log secrets, API keys, proxy credentials, authorization headers, or full cookies.

## Quality gates

- Run Go formatting, static checks, unit tests, and race-sensitive tests where relevant.
- Run frontend lint, type-check, unit tests, and a production build where relevant.
- Add integration tests for filesystem mutations, NFO round trips, provider adapters, migrations, and job recovery.
- Treat path traversal, symlink escape, SSRF, unsafe XML parsing, and concurrent writes as security-critical.
- Update the relevant file in `docs/` whenever public behavior, configuration, deployment, or architecture changes.

## Product priorities

1. Data safety and correct Kodi-compatible NFO/artwork output.
2. Low idle memory and predictable I/O under large libraries.
3. Usable desktop and mobile workflows.
4. Provider extensibility and operational observability.
5. Feature breadth.
