# Delivery roadmap

The plan favors a complete, safe vertical slice over implementing every media type at once. Each phase ends with runnable software and updated documentation.

## Phase 0 — foundation and risk spikes

Deliverables:

- Go module, Vue/TypeScript PWA workspace, local development commands, lint/test/build gates.
- Configuration loader, structured logging, embedded frontend proof, SQLite migrations, health endpoints.
- Container running as non-root on amd64 and arm64.
- Spikes for large-directory walking, SQLite job leasing, NFO round trip, atomic writes on common local/NAS filesystems, and PWA update behavior.
- Architecture decision records for concrete libraries and public license.

Exit criteria:

- One command runs development mode and Compose runs a production build.
- CI can format, lint, test, build, and assemble an image reproducibly.
- Risk-spike results are measured and recorded under `docs/`.

## Phase 1 — movie discovery

Deliverables:

- First-run admin setup, login/logout, secure sessions, and settings shell.
- Source management with container-path allowlisting and permission checks.
- Durable job engine and SSE progress feed.
- Incremental movie scan, filename hints, existing NFO/artwork discovery, library list/detail UI.
- Mobile navigation, list virtualization, loading/error/empty states, theme tokens, PWA manifest.
- Chinese/English locale bundles, user locale preference, browser-locale fallback, and locale-aware dates/numbers.

Exit criteria:

- A large fixture library can be scanned with bounded memory, cancelled, and resumed/restarted safely.
- Read-only libraries are fully browsable while write actions remain disabled.

## Phase 2 — movie scrape and safe write (MVP)

Deliverables:

- Shared outbound HTTP client with proxy, no-proxy, rate limiting, retry, caching, and redaction.
- TMDb search/details/images adapter and Fanart.tv artwork adapter.
- Candidate selection, metadata draft comparison, manual editing, locked fields, artwork picker.
- Kodi movie NFO parser/writer and deterministic fixture tests.
- Change-set preview/apply, conflict checks, safe sidecar writes, audit history.
- Thumbnail service and bounded artwork download jobs.

Exit criteria:

- The end-to-end MVP acceptance criteria in `product-scope.md` pass on amd64 and arm64.
- Restart/failure tests show no truncated NFO and no silent overwrite of user edits.

## Phase 3 — TV shows

Deliverables:

- Show/season/episode model and filename parser.
- TV discovery, matching, episode details, season/episode artwork, and missing-episode report.
- TV/show/episode Kodi NFO round trips and batch scope controls.

Exit criteria:

- Mixed naming fixtures and multi-episode files behave predictably.
- Batch scraping reports partial failures and can retry only failed children.

## Phase 4 — organization and integration

Deliverables:

- Rename template parser with placeholder validation and conditional segments.
- Dry run, collision/case-fold detection, grouped sidecar moves, cross-filesystem safeguards.
- Duplicate and missing-data filters, CSV export, scheduled scans, webhooks, Kodi JSON-RPC sync.

Exit criteria:

- Property/fuzz tests cover path sanitization and template parsing.
- Fault-injection tests cover partial file operations and document recovery behavior.

## Phase 5 — broader media and ecosystem

Deliverables considered in priority order:

- Concert NFO and scraping.
- Music artist/album sidecar support without ID3 mutation.
- Additional providers based on stable official APIs and user demand.
- Multiple users/roles, API tokens, PostgreSQL, external worker mode, HTML exports.

## Cross-cutting work in every phase

- Threat-model update and dependency review.
- Accessibility, responsive layout, keyboard/touch interaction, and browser testing.
- Metrics for duration, queue depth, provider failures, cache behavior, and filesystem errors.
- Migration, backup/restore, proxy, and failure-mode documentation.
- Performance baselines using reproducible fixture libraries.

## Testing strategy

- **Unit**: filename parsing, normalization, merge policy, templates, path validation, provider mapping.
- **Golden/fixture**: NFO parse/write compatibility and stable JSON/API shapes.
- **Integration**: SQLite migrations/jobs, filesystem mutation, proxy routing, provider mock servers.
- **End-to-end**: first-run setup through scrape/save on desktop and mobile viewport.
- **Fault injection**: cancellation, restart, disk-full, permission changes, timeouts, rate limits, and naming collisions.
- **Fuzz/property**: XML input, filenames, path containment, template syntax, and provider payload boundaries.

## Immediate next implementation slice

After this design is accepted:

1. Scaffold Go and Vue workspaces plus reproducible task commands.
2. Implement configuration, logging, SQLite migration, health endpoints, and embedded placeholder UI.
3. Add Dockerfile/Compose and verify non-root amd64/arm64 builds.
4. Implement durable jobs and a simulated scan progress UI.
5. Implement source safety checks and the first read-only movie scan.
