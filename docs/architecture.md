# Architecture and UI conventions

[Usage contract](usage.md) · [Development](development.md) · [Security](security.md) · [Future work](roadmap.md)

MediaGrap is a modular monolith, not a streaming server. Go serves an embedded Vue PWA, HTTP JSON APIs, SSE and persisted workers in one non-root container; optional ffprobe is invoked as a bounded local subprocess. SQLite WAL holds application state, mounted sources hold media, and provider adapters perform outbound requests. No external worker, broker, Node runtime or PostgreSQL adapter is required/provided in production.

```text
Browser/PWA ── HTTP JSON + SSE ── HTTP handlers ── application services
MCP client ── scoped Bearer API ── MCP adapter ── automation/jobs
                                                       │
                                     SQLite repositories + adapters
                                                       │
                               mounted filesystem / TMDb / Fanart.tv
                                                       │
                           transactional outbox ── Webhook deliveries
```

## Source map and boundaries

| Source | Responsibility |
| --- | --- |
| [cmd/mediagrap](../cmd/mediagrap/) and [internal/app](../internal/app/) | Entrypoint, environment/flags, composition, lifecycle, healthcheck, logs |
| [internal/httpapi](../internal/httpapi/) and [internal/auth](../internal/auth/) | Standard `net/http` routes, session/CSRF, response DTOs, embedded UI/pinned external source redirect |
| [internal/library](../internal/library/) | Sources, scanner, sidecar attribution, catalog/TV identity, inspection and browser rename plans |
| [internal/metadata](../internal/metadata/) | Canonical records, saved-metadata priority, provider selection, NFO/artwork orchestration |
| [internal/nfo](../internal/nfo/), [internal/files](../internal/files/), [internal/artwork](../internal/artwork/) | Kodi XML, plan/apply validation, staged publication, safe downloads |
| [internal/providers](../internal/providers/) | TMDb/Fanart.tv adapters; normalized capabilities, not leaked upstream payloads |
| [internal/jobs](../internal/jobs/) | Durable queue, registered handlers, progress, cancellation/retry, SSE state |
| [internal/platform/database](../internal/platform/database/) and [internal/settings](../internal/settings/) | Pure-Go SQLite (`modernc.org/sqlite`), embedded ordered migrations, settings |
| [internal/events](../internal/events/), [internal/webhooks](../internal/webhooks/) | Transactional terminal events, fan-out and separate leased delivery workers |
| [internal/tokens](../internal/tokens/), [internal/mcp](../internal/mcp/), [internal/automation](../internal/automation/) | Hashed source-scoped credentials, SDK adapter, idempotent tasks and browser-approved movie plans |
| [web/src](../web/src/) | Vue 3 Composition API, TypeScript, vue-router, domain API clients/composables and reusable components |

Use behavior-oriented interfaces at the consumer boundary; keep SQL-specific operations visible rather than inventing a universal repository framework. Provider DTOs stay inside adapters. Future database portability is a boundary constraint, not a second implemented database. Current frontend uses composables; Pinia/query-library adoption is not a requirement.

`server.go` is the route registry; source-named handlers and API clients define the current HTTP contract, not an aspirational `/change-sets` or `/metrics` endpoint list. Browser mutations require Session + CSRF except explicit setup/login. MCP tokens do not authorize browser management. Health/source endpoints have deliberate public behavior described in [deployment](deployment.md).

## SQLite and scanning

SQLite uses WAL, foreign keys, a 5-second busy timeout and a **four-connection** bounded pool. Do not weaken durability pragmas to improve benchmark numbers. Embedded SQL migrations are ordered and transactional; automatic backup/restore and disk preflight are not implemented. Consume/close result sets before nested queries or filesystem discovery: an independent SQLite connection succeeding cannot disprove shared-pool starvation.

The scanner has one serialized writer with **two ordered preparation readers**, **200-candidate SQL batches** and directory-scoped snapshots:

1. Enumerate a directory once; snapshot supported sidecar names, process direct files first, then release the listing before descending. Only pending child paths remain on the traversal stack.
2. Readers Lstat media/sidecars and parse filename hints into fixed ordered slots filling the remaining batch space. No goroutine per file or library-wide identity cache.
3. Commit changed media/sidecar/TV rows plus unchanged seen tracking in a SQL-only batch. Filesystem reads and XML parsing do not run inside its write transaction.
4. With IDs known, prepare eligible movie NFO imports in one-record-per-reader windows; input is capped at **1 MiB plus one overflow byte**. Apply in order with an upsert atomically conditional on empty saved title. Racing user/provider saves win, preserving provider/locks. Invalid NFO falls back without rewriting it.
5. Reconcile missing records and completed-scan time only after successful traversal/final hydration/context checks in a final transaction. Failed/cancelled batches roll back locally; earlier commits remain visible without missing reconciliation. Retry starts a fresh traversal/seen epoch.

Fingerprints hash video and attributed sidecar **stat data**, including basename NFO and `movie.nfo` fallback candidate states, not contents. Full scans bypass fingerprints but preserve saved metadata. Successful managed video renames invalidate fingerprints in the path-update transaction so classification changes on the next scan; failed updates do not. Root-layout TV hints and sample/Extras exclusions remain deterministic across batches. [Usage](usage.md#sources-and-scanning) records classification and empty-mount hazards.

Memory depends on the widest directory, pending paths, up to 200 candidates/sidecars and two parsed NFOs, not a strict byte ceiling. Readers join on cancellation/error; an already-running NAS syscall may delay return. Same-size/same-mtime replacements, non-atomic traversal and unmeasured NAS latency remain limits. Local synthetic benchmarks in the source are not production memory/throughput promises.

## Mutations, jobs and integration consistency

Separate discovery, provider search, explicit candidate replacement, manual save and artwork/rename apply. **Browser candidate selection intentionally writes NFO immediately** after validation; do not document it as preview-only. Manual plans retain explicit review, and metadata save can succeed before NFO fails. NFO serialization is deterministic Kodi XML, not preservation of every unknown extension.

File writes validate source boundaries, leaf types, permissions and conflicts before same-directory temporary write/sync/atomic publication where supported, with audit. Cross-device browser rename uses staged copy/size verification; automation per-file rename additionally uses content hashes and recovery records. Filesystem changes and SQLite cannot form a single atomic transaction. Never promise backups, global rollback or identical recovery across browser TV batches/directory moves and automation movie plans.

Jobs persist before execution. One media runner dispatches registered kinds; cancelled state cannot later become succeeded. Startup marks abandoned running jobs interrupted. SSE history persists event IDs and clients reconcile snapshots. Long work must retain bounded progress/cancel/retry semantics; do not silently add workers or leases based on old design diagrams.

Job terminal state, SSE event and external outbox event commit together. Webhook fan-out creates a distinct leased delivery queue; two delivery workers do not occupy media-runner slots. Frozen bodies and duplicate-safe fan-out permit retries, not exactly-once delivery. Outbox backpressure limits record count, not disk bytes.

MCP uses the official SDK and source-scoped services, not HTTP callbacks into REST. Token source grants are normalized and revoked on source deletion even if a numeric ID is reused. Automation movie plans freeze operations, relative paths, metadata/file fingerprints and digest. Browser approval binds token, administrator and digest; consumption and job enqueue are atomic. Shared mutation execution and rooted handles confine writes. Per-operation intent/staged/publishing/done records support startup verification of published results; ambiguous state becomes `needs_review`, never blind replay. [Integration contracts](integrations.md) contain exact scopes/limits.

## Frontend state and routing

The URL is authoritative for primary page, selection and workshop. Browser history/bookmarks/refresh restore location, not unsaved drafts:

| URL example | Selection |
| --- | --- |
| `/movies` | Movie catalog |
| `/movies?movie=123&tab=artwork` | Movie Artwork |
| `/shows?show=20` | Show |
| `/shows?show=20&season=2` | Season |
| `/shows?show=20&season=2&episode=456&tab=nfo` | Episode NFO |
| `/jobs` | Jobs, audit and system status |
| `/settings` | Sources, providers, interface and integrations |
| `/settings?section=renaming` | Rename defaults |

Tabs are `overview`, `artwork`, `cast`, `nfo`, `files`; default overview is omitted. Invalid/incomplete queries are canonicalized; TV specificity is episode → season → show. `/sources` redirects to `/settings` for old bookmarks. Page/selection/workshop changes create history entries; filters/search are not a promised URL contract.

`useLibrary` owns serialized scan tracking: one active page, one recent page and up to eight sequential exact lookups per cycle, with backoff and disposal. Only a 100-row recent ID/state/retry snapshot is retained; no bootstrap replay or max-ID watermark. Catalog and inspector refreshes coalesce on observed terminal transitions, including same-state retries. Explicit active discovery handles old long-running jobs outside recent history.

Inspectors own drafts, availability checks and operation tokens. Scan notifications do not remount them; dirty state, pending saves/dialogs or file/artwork workshops defer replacement. Read-only availability can disable actions without destroying edits. Typed 404/absent episode differs from network/5xx uncertainty. Selection generations reject stale results; settling an obsolete mutation must still release its own busy state. Metadata-saved/NFO-not-written is a real partial result, not generic success. Preserve these constraints when refactoring.

Movie catalog requests are bounded at 50 rows and reject stale generations; DOM rendering uses 56px rows, 4px spacing and five overscan rows either side, lazy/async images and stable ID selection. Previously loaded lightweight records remain in memory. TV keeps its distinct tree model. `JobCenter`, `AuditCenter` and their row/card children own focused presentation state; `useDismissiblePopover` owns shared listener cleanup.

## UI and design tokens

The maintained specification is **the runtime CSS/components plus this section**, not external prototype exports. Preserve [runtime branding](../web/public/assets/), required upstream notices and [main.css](../web/src/assets/main.css). No exported HTML, Google design ID or downloaded mockup is a development dependency.

- Desktop: 64px icon-only navigation rail, 320px media catalog and flexible inspector. Toolbar height 40px with Overview, Artwork, Cast, NFO Raw, File Audit and Save & Write NFO / Scrape actions.
- Mobile: bottom four-tab navigation, catalog→inspector drill-down, sticky workshop tabs/actions. `LibraryWorkspace` owns the **760px** catalog breakpoint; `AppSidebar` bottom navigation uses **768px**. Do not add competing catalog breakpoints. Rename actions reserve `60px + env(safe-area-inset-bottom)` at ≤768px and wrap with 44px touch targets at ≤700px.
- In-app icon: `web/public/assets/logo-icon.png`. Dark slate base `#0c1324`, cards `#191f31`, elevated `#23293c`, tonal layering; keep dark/light/system theme tokens rather than hardcoded new palettes.
- Fonts: Inter for UI; JetBrains Mono for paths/XML/specifications. Current CSS imports Google Fonts remotely with system fallbacks; do not claim bundled/offline fonts.
- Reuse `.btn-*`, `.spec-pill`, `.spec-badge`, `.dot-*`, `.card`, `.tab-*` utilities. Keep focus, loading/empty/error/conflict and localized states visible.
- Audit disclosure uses native buttons with `aria-expanded`/`aria-controls` and Enter/Space behavior. Verify keyboard access, mobile safe areas and no horizontal page overflow, not screenshots alone.
- English/Chinese locale keys belong in shared locale logic. UI locale is separate from provider metadata language. Render metadata as untrusted text, not executable markup.

The service worker caches the shell/static assets, not safe offline editing or the large `/source` archive. Build provenance, dependency source/notices and source-download behavior are defined by [distribution](distribution.md), not runtime scrape caches.
