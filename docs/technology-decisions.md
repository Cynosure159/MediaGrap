# Technology decisions

## Decision summary

| Area | Choice | Reason |
| --- | --- | --- |
| Backend | Go | Small static binary, low idle overhead, strong concurrency and networking, fast filesystem traversal, simple cross-compilation |
| HTTP | Go standard library plus a small router | Keeps dependency and runtime cost low while preserving middleware and typed handler structure |
| Frontend | Vue 3 + TypeScript + Vite | Compact reactive UI, strong mobile ergonomics, mature PWA tooling, and maintainable component model |
| State/data | Pinia + a query/cache library | Separate local UI state from server state, retries, invalidation, and optimistic editing |
| Database | SQLite with WAL and migrations | Zero-configuration single-container persistence, transactional state, good read concurrency |
| API | Versioned JSON REST plus Server-Sent Events | Simple CRUD/control plane; SSE is sufficient for one-way job and library updates |
| Metadata | Provider adapters, TMDb first | Prevents provider schemas and policies from becoming the domain model |
| Media inspection | Optional `ffprobe` subprocess | Mature codec/container inspection without embedding a large media stack in the application |
| Packaging | Multi-stage Docker build, embedded frontend | One runtime process and no Node.js toolchain in production |
| Observability | Structured logs, health/readiness, metrics endpoint | Useful in Docker without requiring an external stack |

## Why Go rather than Rust

Both languages meet the performance goal. Go is recommended for this product because the workload is dominated by filesystem traversal, HTTP, XML/JSON transformation, image download, and orchestration rather than CPU-bound parsing. Go offers a shorter path to a maintainable service, easy static cross-builds, built-in concurrency primitives, and operationally simple profiling.

Rust remains a valid option if the team already has strong Rust expertise or later needs a memory-safe native parser with tighter peak-memory control. It would likely increase initial implementation time and library-integration complexity without materially improving normal scrape latency, which is dominated by network and disk.

## Modular monolith before microservices

The initial application is one deployable process but is divided into domain modules and adapter interfaces. This avoids network boundaries, multiple databases, and orchestration overhead on home servers. The job executor can later become a separate process because jobs, leases, and provider interfaces are explicit from the start.

## SQLite rules

- Store the database under the persistent config/data volume, never beside media files.
- Enable WAL, foreign keys, a busy timeout, and bounded connection counts.
- Serialize migrations and filesystem change-set commits.
- Store canonical metadata, provider identifiers, file fingerprints, job state, settings, audit events, and caches with explicit expiry.
- Do not store downloaded artwork blobs in SQLite; store files in cache or alongside media and keep references/checksums in the database.
- Abstract repositories at domain boundaries, but do not introduce a generic repository layer that hides useful SQL.

## Frontend principles

- Use responsive master/detail layouts: a compact bottom navigation on phones and a sidebar on larger screens.
- Virtualize large grids/lists and request paginated data.
- Serve thumbnails sized for the viewport; never send original artwork into a library grid.
- Cache only the application shell and safe static assets in the service worker. Do not pretend metadata edits are offline-safe.
- Show connection, job, conflict, dirty-edit, and application-update states explicitly.
- Meet WCAG 2.2 AA contrast/focus/touch-target expectations where practical.

## Explicitly deferred decisions

- Exact router, SQL helper/code generator, component library, query library, and migration tool will be selected during the foundation spike using small proof-of-concept benchmarks.
- OpenAPI generation direction will be chosen after the first three endpoints establish request/response patterns.
- The final public license and name require owner approval before publishing.

