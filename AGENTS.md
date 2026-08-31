# MediaGrap repository instructions

## Scope

MediaGrap is a lightweight, self-hosted media metadata manager. The target production shape is a single Docker container containing a Go server and an embedded Vue PWA.

## Required workflow

- Read `README.md`, `README-zh.md`, and the relevant design documents before changing architecture or behavior.
- Prefix terminal commands with `rtk`, as required by `/Users/cy/.codex/RTK.md`.
- Keep detailed human-facing project documentation under `docs/`. Root `README.md`, `README-zh.md`, and `AGENTS.md` are the only exceptions.
- Preserve unrelated user changes and never rewrite media files without an explicit, previewable operation.
- Prefer small, reviewable commits and conventional commit messages.

## UI/UX & Design System Constraints (Mandatory)

All frontend feature development, page layouts, component design, and responsive behaviors MUST strictly follow the exported Stitch design prototypes and design tokens:
- **Local Prototypes Reference**: `docs/stitch-prototypes/` (Contains HTML and PNG screenshots for all 10 desktop & mobile screens).
- **Design Specification Document**: `docs/ui-ux-design.md`.
- **Layout Architecture**: 
  - Desktop: 3-column high-density split pane (64px fixed left Nav Rail + 320px Media Catalog List + Flex-1 Inspector Workspace).
  - Inspector Toolbar: 40px top bar with 5 standard workshop tabs (`Overview`, `Artwork`, `Cast`, `NFO Raw`, `File Audit`) and quick action buttons (`Save & Write NFO`, `Scrape`).
  - Mobile: Native bottom 4-tab bar, sticky top workshop tabs, and floating bottom action bar.
- **Brand & Visuals**:
  - Use the icon-only geometric mark (`web/public/assets/logo-icon.png`) for in-app headers and rails.
  - Deep dark slate theme (`#0c1324` base, `#191f31` cards, `#23293c` elevated) with tonal layering elevation.
  - Typography: `Inter` for UI text, `JetBrains Mono` for tech specs, resolution pills, NFO XML, and filesystem paths.
  - Reusable CSS utilities: `.btn-*`, `.spec-pill`, `.spec-badge`, `.dot-*`, `.card`, `.tab-*` defined in `web/src/assets/main.css`.

## Architecture constraints

- Backend: Go, organized as a modular monolith with explicit domain and adapter boundaries.
- Frontend: Vue 3, TypeScript, Composition API, responsive and installable as a PWA.
- Default database: SQLite in WAL mode. Keep repository interfaces portable enough for a future PostgreSQL adapter.
- Production: one non-root container and one application process; embed compiled web assets in the Go binary.
- Filesystem writes must use plan/preview, path validation, conflict detection, temporary-file-plus-atomic-rename where supported, and an audit record.
- Scraper integrations must sit behind provider interfaces. Do not leak provider payloads into domain models.
- Long-running work must use persisted jobs with cancellation, retry policy, progress, and structured logs.
- Never log secrets, API keys, proxy credentials, authorization headers, or full cookies.
- Add structured logs at diagnostic boundaries: external-provider request start/result/failure, persisted-job lifecycle transitions, and filesystem plan/apply outcomes. Logs must include safe correlation fields (for example media item ID, provider endpoint, HTTP status, duration, or result count) while excluding secrets and sensitive request URLs.

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
