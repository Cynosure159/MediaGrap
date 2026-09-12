# Phase 5A — Jobs and operations visibility

Phase 5A's first usable slice adds an authenticated operations center backed only by persisted server data. It does not rename, move, delete, restore, or otherwise mutate media files.

## Durable jobs

- `GET /api/v1/jobs` returns the latest 100 persisted jobs with queue, progress, retry, and lifecycle timestamps.
- `GET /api/v1/jobs?activeScans=true&after=<id>` returns at most 100 queued/running scan jobs ordered by ascending ID, with `nextCursor` (zero when exhausted). Negative/invalid cursors return 400. This bounded discovery page does not alter the default recent-history response.
- `GET /api/v1/jobs/{id}` retrieves an exact persisted job (400 for invalid IDs, 404 when unavailable), including terminal scans older than recent history. Both read endpoints require a session and never return private job payloads.
- `POST /api/v1/jobs/{id}/cancel` accepts queued and running jobs. A queued job is cancelled before pickup; a running job also receives context cancellation so cooperative scan/download handlers stop.
- `POST /api/v1/jobs/{id}/retry` requeues failed, cancelled, or interrupted work using its original persisted payload. Other states return a conflict.
- `GET /api/v1/jobs/events` is an authenticated SSE stream. State and progress events are persisted in `job_events`, carry monotonic event IDs, and support browser `Last-Event-ID` reconnection. The frontend also refreshes the job snapshot periodically so an expired or interrupted stream cannot leave permanent stale state.
- Worker startup still marks abandoned `running` jobs as `interrupted`; file mutation jobs are never silently resumed halfway through an operation.

The library uses one serialized polling scheduler with active-page discovery and bounded exact lookups rather than per-scan 60-second loops. It survives navigation/reload, refreshes catalogs/inspectors on all terminal transitions, backs off request errors, and aborts polling on unmount. See [bounded readers and persisted scan tracking](scan-readers-and-tracking.md) for intervals, limits, measurements and browser evidence.

Cancellation is cooperative. A filesystem or network primitive that is already completing may finish before its context check, but a cancelled database state is never subsequently overwritten with `succeeded`.

## Audit and recovery visibility

`GET /api/v1/audit-entries` returns the latest 200 NFO/artwork preview and apply records. Absolute stored paths are reduced to a safe parent/basename display before leaving the server. The response distinguishes previewed and applied outcomes and states the currently supportable recovery boundary.

The current safe-write engine provides same-directory temporary files and atomic replacement where the filesystem supports it. It does not yet create durable backup copies, so the UI explicitly describes applied writes as `atomic_replace_only` and does not promise global rollback after NAS or storage failure.

## System status

`GET /api/v1/operations/status` is authenticated and reports:

- application version, commit, and build time;
- database readiness, latest migration, SQLite WAL mode, and database size;
- cache availability, writability, and current file usage;
- configured media-source availability, writability, and indexed item count;
- whether TMDb, Fanart.tv, and the outbound proxy are configured, without returning credentials or proxy values.

Provider rows distinguish configuration readiness from real connectivity. Authenticated fixed-target tests for TMDb, Fanart.tv, and the configured proxy persist a redacted latest result with HTTP status and duration. System status also reads the active SQLite journal mode instead of assuming WAL and exposes each source's scan policy and schedule timestamps.

## Completed settings

- Primary and fallback metadata language are persisted; TMDb detail requests fill missing translated title/overview fields from the fallback language.
- One outbound HTTP/HTTPS/SOCKS5 proxy plus explicit `NO_PROXY` rules is shared by provider and artwork traffic.
- Source policies support full or incremental scans and restart-safe interval schedules. Duplicate scheduled scans are suppressed while a source already has queued/running work.
- Theme and interface language are stored per authenticated user; browser storage remains only an immediate-render cache.
- Credentials, proxy URLs, proxy credentials, and NO_PROXY contents are write-only from the API perspective.

## UI

The Jobs navigation item opens the responsive Operations page. `OperationsPage` composes three focused sections:

- `JobCenter` for active progress, cancellation, retry, history, and SSE connection state;
- `AuditCenter` for write plans/results and recovery scope;
- `SystemStatusPanel` for container, database, cache, mount, provider, and proxy state.

At mobile widths the sections become a single-column flow above the existing bottom navigation. All visible values come from authenticated APIs; unavailable data is never replaced with prototype success text.

## 0.0.2 database starvation protection

See [database diagnostics and validation](database-diagnostics.md) for shared-pool readiness, numeric connection statistics, bounded probes, atomic scan deduplication, catalog connection-lifetime fixes, and sidecar I/O improvements.

## Remaining Phase 5A work

- Backup creation/retention and restore operations once their file safety plan/apply boundary is implemented.
- Parent/child batch jobs and retrying only failed children for TV batch operations.
