# MediaGrap

**English** | [简体中文](README-zh.md)

MediaGrap is a lightweight, self-hosted media metadata scraper and library manager for NAS devices, home servers, and other Docker environments. It provides a responsive Web UI for managing metadata, NFO files, and artwork for movies, TV shows, and other media.

The project aims to retain the essential workflows found in tools such as tinyMediaManager and MediaElch while reducing server-side resource usage and providing a better browser, mobile, and container experience.

> Current status: product and architecture planning. Application code has not yet been implemented.

## Goals

- Use Go for a resource-efficient backend with strong filesystem and concurrency performance.
- Provide a clean, responsive Vue 3 Web UI.
- Support mobile browsers and PWA installation.
- Use a single Docker container as the standard deployment model.
- Support linux/amd64 and linux/arm64.
- Support HTTP, HTTPS, and SOCKS5 proxies with `NO_PROXY` rules.
- Run scans, scraping, artwork downloads, and file changes as observable background jobs.
- Read and write Kodi-compatible NFO and artwork safely.
- Preview and validate every rename, move, and overwrite operation before execution.

## Technology

| Area | Choice |
| --- | --- |
| Backend | Go modular monolith |
| Frontend | Vue 3, TypeScript, Vite |
| Web application | Responsive UI, PWA, light and dark themes |
| Database | SQLite WAL, with boundaries for a future PostgreSQL adapter |
| API | Versioned JSON REST API and SSE job events |
| Metadata | Extensible provider adapters, beginning with TMDb |
| Artwork | TMDb first, with optional Fanart.tv support |
| Media inspection | Optional `ffprobe` integration |
| Deployment | Multi-stage build, non-root container, frontend embedded in the Go binary |

Go is preferred over a Java server runtime to keep idle memory use low and deployment simple. The expected workload is dominated by directory traversal, network requests, XML/JSON processing, image downloads, and filesystem writes, where Go offers a good balance of performance and implementation complexity.

## Planned features

### MVP: movies

- First-run administrator setup, authentication, and secure sessions.
- Management of media directories mounted into the container.
- Full and incremental scans, filename parsing, and existing NFO/artwork discovery.
- TMDb search, details, and image scraping.
- Candidate selection, metadata editing, and protection of manual fields.
- Kodi movie NFO reading and writing.
- Poster, fanart, logo, and other artwork selection and download.
- Persistent jobs with progress, cancellation, retry, and restart recovery.
- File change previews, safe writes, backup policy, and audit history.
- Docker Compose deployment, health checks, and proxy configuration.

### Later releases

- TV show, season, and episode management with missing-episode detection.
- Template-based renaming, conditional expressions, collision checks, and dry runs.
- Duplicate and missing-metadata filters.
- CSV export, scheduled scans, webhooks, and Kodi JSON-RPC synchronization.
- Concert and music sidecar metadata.
- Additional providers, multiple users, API tokens, PostgreSQL, and external workers.

## Architecture

```text
Desktop browser / mobile PWA
             |
        HTTP JSON + SSE
             |
┌──────────────────────────────────────────┐
│                MediaGrap                 │
│                                          │
│ Web/API  Auth  Library  Metadata  Jobs   │
│                   |                      │
│       Application services / domain      │
│              |            |              │
│           SQLite      adapter APIs       │
│                         |      |          │
│                    filesystem providers  │
└──────────────────────────────────────────┘
              |               |
       config/data volume  mounted media
                              |
                    TMDb / Fanart.tv / ...
```

The production build embeds the Vue frontend in the Go binary, allowing the Web UI, API, and background workers to run as one process. The initial modular-monolith design avoids the operational and resource cost of microservices while retaining boundaries that allow workers to be separated later if needed.

## Filesystem safety

Media libraries contain valuable user data, so safety takes priority over feature breadth:

1. Scanning only updates the index; it does not automatically scrape or modify files.
2. Writes and renames first produce an immutable change plan.
3. Paths, permissions, source boundaries, and conflicts are revalidated before execution.
4. NFO data is written to a temporary file in the target directory before atomic replacement.
5. Existing destinations are not overwritten by default.
6. Cross-filesystem moves use copy, verification, and only then source removal.
7. Every mutation produces an audit event without exposing secrets.

## Docker deployment model

Planned container paths:

| Container path | Purpose | Access |
| --- | --- | --- |
| `/config` | SQLite database, configuration, keys, jobs, and audit state | Read/write |
| `/cache` | Provider cache, thumbnails, and temporary downloads | Read/write; disposable |
| `/media/...` | Explicitly mounted media libraries | Read-only or read/write |

Release images will run as a non-root user and target amd64 and arm64. Administrators explicitly mount host media paths into the container; the Web UI will not provide unrestricted host filesystem browsing.

## Roadmap

1. Scaffold Go, Vue PWA, SQLite migrations, test infrastructure, and container builds.
2. Implement authentication, source management, durable jobs, and read-only movie scanning.
3. Integrate TMDb, metadata editing, Kodi NFO, and safe writes to deliver the MVP.
4. Add TV shows, seasons, and episodes.
5. Add renaming, exports, scheduled jobs, and Kodi synchronization.
6. Expand to music, concerts, more providers, and multi-user operation as demand requires.

See the detailed [delivery roadmap](docs/roadmap.md).

## Documentation

- [Product scope and acceptance criteria](docs/product-scope.md)
- [System architecture](docs/architecture.md)
- [Technology decisions](docs/technology-decisions.md)
- [Docker and deployment design](docs/deployment-design.md)
- [Delivery roadmap](docs/roadmap.md)
- [MediaElch reference review](docs/mediaelch-reference.md)
- [Phase 0 foundation](docs/phase-0-foundation.md)
- [ADR 0001: Foundation stack](docs/adr/0001-foundation-stack.md)

## MediaElch reference boundary

MediaGrap may learn from MediaElch's public features, workflows, Kodi NFO compatibility, and architectural boundaries. It will not directly copy LGPL-3.0 source code, tests, icons, translations, or other protected assets.

Provider integrations will prefer official APIs and independently review API key, rate-limit, caching, attribution, and branding requirements. Kodi NFO support will be implemented from public format documentation and independently authored fixtures.

## Documentation convention

Detailed project documentation belongs under `docs/`. The repository entry points `README.md`, `README-zh.md`, and the collaboration instructions `AGENTS.md` are the only root-level exceptions. Changes to public behavior, configuration, deployment, or architecture must update the relevant documentation.

## License

The public project license has not yet been selected. License selection and dependency compliance review are required before publishing or incorporating third-party code.
