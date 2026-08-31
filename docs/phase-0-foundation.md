# Phase 0 foundation

## Delivered baseline

- Go application with environment/flag configuration, structured logging, graceful shutdown, and health/readiness endpoints.
- SQLite database opened in WAL mode, with foreign keys, busy timeout, embedded SQL migrations, and a first schema migration.
- Vue 3 + TypeScript application shell using Composition API and typed component boundaries.
- Installable PWA manifest and service-worker registration. The service worker caches only the application shell and static assets; API mutations are not cached offline.
- Frontend assets can be embedded in the Go binary for the production image.
- Multi-stage Dockerfile, Compose configuration, health check, non-root runtime, and initial GitHub Actions workflow.

## Local commands

Prerequisites: Go 1.26+, Node 24.15+, and npm 11+.

```bash
npm --prefix web install
go mod download
make dev
```

`make dev` runs the Go API on `http://127.0.0.1:8080` and Vite on its printed development URL. Vite proxies API requests to the Go server. The local database and cache are created under `.local/`, which is ignored by Git.

Useful verification commands:

```bash
make test
make typecheck
make build
docker compose up --build
```

`make build` builds the Vue application, copies the static output into the ignored Go embed directory, and writes the binary to `bin/mediagrap`. Fresh source builds serve a clear fallback page until this step has run.

## Current API contract

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Liveness probe; does not query SQLite |
| `GET /readyz` | Readiness probe; checks the SQLite connection |
| `GET /api/v1/system/info` | Build identity for the UI and diagnostics |
| `GET /api/v1/system/summary` | Foundation dashboard summary; placeholder counts until Phase 1 |

## Vue component map

| Component | Responsibility | Contract |
| --- | --- | --- |
| `App.vue` | Composes the shell and coordinates the system summary request | owns selected navigation state; passes data downward |
| `AppSidebar.vue` | Renders responsive desktop/sidebar or mobile/bottom navigation | `items`, `activeSection`; emits `selectSection` |
| `SystemPulse.vue` | Shows availability and high-level server counts | summary/loading/error props; emits `refresh` |
| `ActivityRail.vue` | Explains completed and pending foundation work | `summary` prop only |
| `useSystemSummary.ts` | Fetches and cancels system-summary requests | exposes state and `refresh()` |

The visual system is intentionally based on a server control room rather than a media-streaming catalogue: graphite-blue service surfaces, sea-glass status tones, and one amber activity accent. The activity signal in the service panel is the signature element; surrounding layouts stay quiet and information-dense.

## Risk spikes still required

The following need measurement against target filesystems or browser devices before Phase 1 begins:

- directory walk performance and memory bounds on a 10,000+ file fixture;
- SQLite job lease/recovery semantics under abrupt process termination;
- NFO write/rename atomicity on supported NAS filesystems;
- PWA install/update behavior on Android Chrome and iOS Safari.

The code foundation intentionally does not claim these results until measured.
