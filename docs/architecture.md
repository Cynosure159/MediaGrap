# Architecture

## Context

MediaGrap runs close to the user's media files, usually as a Docker container on a NAS or home server. Browsers communicate with the application over HTTP(S), while the application accesses metadata providers through an optional proxy and reads/writes only configured media sources.

```text
Desktop / mobile PWA
        |
   HTTP JSON + SSE
        |
┌─────────────────────────────────────────────────────────┐
│ MediaGrap process                                      │
│                                                       │
│ Web/API  Auth  Library  Metadata  Files  Jobs  Admin   │
│    |      |      |         |        |      |      |    │
│    └──────── application services / domain ─────────┘   │
│                 |                 |                     │
│          SQLite repositories   adapter interfaces      │
│                                   |          |          │
│                              filesystem   providers     │
└─────────────────────────────────────────────────────────┘
          |                         |
 config/data volume          mounted media roots
                                    |
                          TMDb / Fanart.tv / future APIs
```

## Suggested repository layout

```text
cmd/mediagrap/             application entry point
internal/
  app/                     composition, lifecycle, configuration
  auth/                    users, sessions, password policy
  library/                 sources, scan/index, media identity
  metadata/                canonical models, merge/edit policy
  nfo/                     Kodi XML read/write and compatibility
  artwork/                 selection, download, transform/cache
  files/                   change sets, rename/move, safe writes
  jobs/                    durable queue, leases, progress, events
  providers/               provider contracts and registrations
  adapters/
    sqlite/                persistence and migrations
    filesystem/            OS/container filesystem implementation
    tmdb/                  TMDb adapter
    fanarttv/              Fanart.tv adapter
  httpapi/                 routes, handlers, middleware, SSE
web/                       Vue/TypeScript PWA source
migrations/                embedded database migrations
deploy/                    Docker and Compose assets
docs/                      all human-facing documentation
```

Packages under `internal/` should expose behavior-oriented interfaces at the point of use. Provider-specific DTOs remain inside their adapter.

## Component responsibilities

### HTTP and authentication

- Serve embedded frontend assets and `/api/v1` endpoints.
- Use secure, HTTP-only, SameSite cookies for browser sessions and CSRF protection for mutations.
- Bootstrap the first administrator only when no user exists.
- Apply request size limits, timeouts, rate limiting on authentication, and structured request IDs.
- Expose `/healthz` for process health and `/readyz` for database/migration readiness.

### Library scanner

- Walk configured sources with bounded concurrency and cancellable contexts.
- Match allowed media extensions and ignore patterns; detect DVD/Blu-ray structures without treating internal directories as separate titles.
- Define a symlink policy per source. The safe default is not to follow directory symlinks.
- Capture cheap stat fingerprints first; compute content hashes only when identity is ambiguous.
- Parse filenames into hints but never rename during scanning.
- Reconcile discovered assets transactionally and mark missing items instead of immediately deleting records.

### Metadata domain

- Keep a provider-neutral canonical representation of titles, dates, ratings, IDs, people, studios, genres, certification, plot, artwork, and media details.
- Track field provenance and user overrides. A rescrape must not silently overwrite locked/manual fields.
- Treat search candidates separately from committed metadata.
- Validate types and limits before persistence and NFO generation.

### Provider adapters

Each provider implements only the capabilities it supports, such as search, details, images, episodes, or people. Requests flow through a shared network client that provides:

- global/per-provider proxy selection and no-proxy rules;
- connect/request timeouts and cancellation;
- provider-specific concurrency and token-bucket rate limits;
- bounded retry with jitter for retryable failures and `Retry-After` support;
- response-size limits, caching, secret redaction, and metrics;
- SSRF protection for user-influenced URLs.

Provider terms, attribution, API key requirements, and cache rules must be documented before enabling an adapter.

### NFO subsystem

- Parse existing Kodi-style XML while preserving unsupported fields when feasible.
- Generate deterministic UTF-8 XML with tests against fixtures.
- Version compatibility behavior rather than scattering Kodi-version checks.
- Write a temporary file in the target directory, flush/close it, optionally back up the previous sidecar, then atomically rename where the filesystem supports it.
- Never use XML external entities or unrestricted entity expansion.

### File change engine

Filesystem mutation is a two-stage protocol:

1. **Plan**: normalize and validate paths, resolve source boundaries, calculate destination names, check permissions/collisions/case-folding, and persist an immutable change set.
2. **Apply**: acquire item/source locks, revalidate preconditions, execute ordered operations, record results, and invalidate the library index.

Rules:

- A configured source root is a capability boundary; resolved paths must remain within it.
- Never accept an arbitrary host path from an API mutation.
- Avoid overwrite by default. Cross-filesystem moves require copy, verification, and only then source removal.
- Rename plans include media and selected sidecars as one group.
- The audit event records actor, intent, paths, checksums where useful, and outcome without secrets.
- Full rollback cannot be guaranteed for arbitrary NAS failures, so the UI must state exactly what is recoverable.

### Job system

Jobs are stored in SQLite before execution. The first implementation uses in-process workers with bounded queues.

States: `queued`, `running`, `succeeded`, `failed`, `cancelled`, and `interrupted`. Workers lease jobs and update heartbeats. On startup, expired `running` leases become `interrupted`; idempotent jobs may be retried automatically while file mutations require precondition revalidation.

Parent/child jobs represent batches. Progress counts units and bytes where known rather than estimating time. SSE publishes durable job snapshots plus transient progress events; clients reconnect with an event cursor and refresh state if history has expired.

## Key workflows

### Scan

```text
User -> create scan job -> walker -> classify assets -> reconcile database
     <- job/SSE progress <- workers <- batched filesystem results
```

Scanning does not call external providers by default. Discovery and scraping remain separate operations so a rescan is predictable and cheap.

### Scrape and save

```text
search provider -> choose candidate -> fetch normalized draft
       -> review/merge fields and artwork -> create change set
       -> preview -> apply safe writes -> update index and audit
```

### Rename

```text
select items -> render naming template -> sanitize -> detect conflicts
       -> show old/new plan -> confirm -> revalidate -> apply -> rescan paths
```

## Concurrency and consistency

- Use bounded worker pools per workload: scan/stat, provider HTTP, image downloads, and mutations.
- Enforce one active mutation per source and an item-level lock to avoid concurrent save/rename races.
- Use optimistic version fields for metadata edits; return a conflict instead of last-write-wins.
- Keep database transactions short and never hold one open during provider requests or large file copies.
- Make job handlers idempotent where possible and use stable operation keys for retries.

## Data model outline

- `users`, `sessions`
- `sources`, `source_scan_cursors`
- `media_items`, `media_files`, `sidecar_assets`
- `external_ids`, `metadata_values`, `people`, `credits`, `ratings`
- `artwork_candidates`, `artwork_assets`
- `jobs`, `job_events`, `job_dependencies`
- `change_sets`, `file_operations`, `audit_events`
- `provider_configs`, `provider_cache`
- `settings`, `schema_migrations`

The detailed schema should follow use cases and migrations rather than being finalized up front.

## API outline

- `/api/v1/session`, `/api/v1/setup`
- `/api/v1/sources`, `/api/v1/sources/{id}/scans`
- `/api/v1/media`, `/api/v1/media/{id}`
- `/api/v1/media/{id}/searches`, `/api/v1/media/{id}/drafts`
- `/api/v1/change-sets`, `/api/v1/change-sets/{id}/apply`
- `/api/v1/jobs`, `/api/v1/events`
- `/api/v1/providers`, `/api/v1/settings/network/test`
- `/api/v1/system/info`, `/healthz`, `/readyz`, `/metrics`

Mutating endpoints use idempotency keys where duplicate submission would be harmful. Errors use a stable machine code, localized-safe message, request ID, and optional field details.

## Threat model highlights

- Path traversal and symlink escape into host mounts.
- SSRF through artwork/provider URLs and malicious redirects.
- XML attacks through crafted NFO files.
- Stored XSS from external metadata rendered in the UI.
- Secret disclosure through settings endpoints, logs, exports, or proxy URLs.
- Cross-site request forgery and weak first-run exposure.
- Resource exhaustion from huge libraries, archive-like images, provider payloads, or unbounded jobs.

Security tests and limits are part of each feature, not a final hardening phase.

