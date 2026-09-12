# MediaGrap repository instructions

## Scope

MediaGrap is a lightweight, self-hosted media metadata manager. The target production shape is a single Docker container containing a Go server and an embedded Vue PWA.

## Required workflow

- Read `README.md`, `README-zh.md`, and the relevant maintained guides before changing architecture or behavior. Start with [development](docs/development.md) and [architecture](docs/architecture.md).
- Use standard Go/npm/Docker commands; RTK is optional for contributors and requires no developer-home configuration.
- Keep detailed human-facing documentation under `docs/`; update canonical guides rather than adding phase diaries. Root `README.md`, `README-zh.md`, `AGENTS.md`, and conventional `LICENSE` are exceptions; short `.github` contribution/security discovery pointers link to `docs/`.
- Preserve unrelated user changes and never rewrite media files without an explicit, previewable operation.
- Prefer small, reviewable commits and conventional commit messages.

## UI/UX & Design System Constraints (Mandatory)

Maintain the existing application design tokens, responsive behavior, and reusable components. Exported prototypes are not a development dependency. Preserve runtime branding/styles and all required upstream legal notices.

- **Maintained specification**: [Architecture UI conventions](docs/architecture.md#ui-and-design-tokens), runtime `web/src/assets/main.css` and reusable components.
- **Layout Architecture**:
  - Desktop: 3-column high-density split pane (64px fixed left Nav Rail + 320px Media Catalog List + Flex-1 Inspector Workspace).
  - Inspector Toolbar: 40px top bar with 5 standard workshop tabs (`Overview`, `Artwork`, `Cast`, `NFO Raw`, `File Audit`) and quick action buttons (`Save & Write NFO`, `Scrape`).
  - Mobile: Native bottom 4-tab bar, sticky workshop tabs and safe-area-aware action bars. The workspace owns the 760px catalog breakpoint; navigation uses 768px. Preserve runtime behavior rather than imposing a prototype breakpoint.
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
- Long-running work must use persisted jobs with cancellation, retry policy, progress, and structured logs. Preserve bounded scanning (two readers, 200-candidate SQL batches), short transactions and SQLite durability.
- Saved metadata wins over scan-time NFO imports; only successful scans reconcile missing files. Preserve dirty drafts, selection-generation guards and missing/uncertain-target write gating during automatic refresh.
- Browser candidate selection currently replaces metadata and writes NFO immediately after validation; do not silently change this or describe it as read-only. Do not claim automatic backups/global rollback or equate browser and automation recovery guarantees.
- Never log secrets, API keys, proxy credentials, authorization headers, or full cookies.
- Add structured logs at diagnostic boundaries: external-provider request start/result/failure, persisted-job lifecycle transitions, and filesystem plan/apply outcomes. Logs must include safe correlation fields (for example media item ID, provider endpoint, HTTP status, duration, or result count) while excluding secrets and sensitive request URLs.

## Quality gates

- Fail on Go formatting differences; run `go vet` and unit/race tests where relevant.
- Run frontend typecheck, unit tests and a production build where relevant. No standalone frontend lint script exists; do not claim one ran. See [actual checks](docs/development.md#checks).
- Add integration tests for filesystem mutations, NFO round trips, provider adapters, migrations, and job recovery.
- Treat path traversal, symlink escape, SSRF, unsafe XML parsing, and concurrent writes as security-critical.
- Update the relevant file in `docs/` whenever public behavior, configuration, deployment, or architecture changes.
- Test with owned temporary/synthetic fixtures, not real libraries or credentials. Follow [security reporting](docs/security.md) and [distribution requirements](docs/distribution.md); preserve upstream notices and matching-build source availability. Dirty snapshots are test-only, not releases.

## Product priorities

1. Data safety and correct Kodi-compatible NFO/artwork output.
2. Low idle memory and predictable I/O under large libraries.
3. Usable desktop and mobile workflows.
4. Provider extensibility and operational observability.
5. Feature breadth.
