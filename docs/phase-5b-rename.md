# Phase 5B — File rename and move

Phase 5B adds a plan/preview/execute rename engine for movies (and a foundation for TV shows). It extends the read-only naming dry-run from Phase 4D into a full durable rename workflow with directory restructuring, sidecar grouping, conflict detection, and audit records.

## API endpoints

- `POST /api/v1/media/{id}/rename-plans` accepts `{ "pattern": "${title} (${year})" }` and returns a persisted `RenamePlan` containing per-file items, conflict status, and a plan ID. The pattern may include `/` to express directory restructuring (e.g. `${title} (${year})/${title} (${year})`). The endpoint requires authentication and a CSRF token.
- `POST /api/v1/tv/shows/{id}/rename-plans` accepts `{ "pattern": "...", "seasonNumber": 1, "episodeId": 123 }` for whole-show, single-season, or single-episode batch renaming.
- `GET /api/v1/rename-plans/{id}` returns a previously created rename plan.
- `POST /api/v1/rename-plans/{id}/apply` queues a durable background job of kind `rename_execute` and returns the job immediately with HTTP 202. The job executes the plan asynchronously with per-item progress, cancellation support, and SSE events.

## Rename plan model

Plans are persisted in the `rename_plans` table with a UUID primary key. Each plan stores:

- `media_item_id` or `tv_show_id`
- `pattern` — the original template string
- `state` — `previewed`, `applied`, `partial`, `failed`, or `cancelled`
- `items_json` — serialized array of `RenamePlanItem` with kind, currentPath, plannedPath, operation, conflict, and status
- `has_conflicts` — whether any item has a detected conflict

## Token resolution

### Movie Tokens
`${title}`, `${originalTitle}`, `${year}`, `${resolution}`, `${videoCodec}`, `${audioCodec}`, `${edition}`, `${imdbId}`.

### TV Show & Episode Tokens
`${showTitle}`, `${seasonNumber}`, `${seasonNumberPad}`, `${episodeNumber}`, `${episodeNumberPad}`, `${episodeTitle}`, `${year}`, `${resolution}`, `${videoCodec}`, `${audioCodec}`.

Token values are resolved from `media_metadata` / `tv_metadata` / `tv_episode_metadata` with fallback to title hints and year hints. Video codec and resolution come from the ffprobe inspection cache. Unknown tokens are rejected.

## Standard Presets

### TV Shows & Episodes
- **Kodi Standard**: `${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad}`
- **Kodi (with Title)**: `${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}`
- **Plex Standard**: `${showTitle} (Season ${seasonNumberPad})/${showTitle} - s${seasonNumberPad}e${episodeNumberPad} - ${episodeTitle}`
- **Jellyfin / Emby Standard**: `${showTitle}/Season ${seasonNumberPad}/${showTitle} S${seasonNumberPad}E${episodeNumberPad} ${episodeTitle}`
- **Flat (No folder)**: `${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}`

## Directory restructuring

When the pattern contains `/`, it is split at the last separator into a directory template and a filename template. Each segment is independently sanitized via the existing `sanitizeFilename` function. If the resolved directory differs from the current parent directory, a `rename_dir` item is added to the plan.

Directory renames are validated against the source root boundary and executed before individual file renames.

## Sidecar grouping

The rename engine uses the same file discovery rules as Phase 4D's `auditFiles`: same-basename companion files (`.nfo`, `.jpg`, `.srt`, etc.) and standard Kodi directory-level assets (when the directory contains exactly one video) are included in the rename plan. Language and edition suffixes are preserved.

## Conflict detection

- **Filesystem conflicts**: each planned destination is checked with `os.Stat`; existing files that are not the source file itself are flagged as conflicts.
- **Case-insensitive collisions**: planned paths are compared via `strings.EqualFold` and tracked in a case-folded map to detect collisions on case-insensitive filesystems.
- Plans with any conflicts have `hasConflicts=true` and cannot be applied.

## Execution safety

At apply time, each item is re-validated:

- Source must still exist and be within the media root boundary (`resolvedParentWithinRoot`)
- Destination must not exist (no implicit overwrite)
- Parent directories are created with `os.MkdirAll`

For same-filesystem renames, `os.Rename` is used atomically. For cross-filesystem moves, the engine uses copy → size verification → rename temp to target → delete source. Cross-device detection uses `syscall.Stat_t.Dev` comparison.

If any item fails, subsequent items continue processing. The plan state is updated to `partial` and each item records its individual `status` (success/failed/skipped).

After all renames complete, the engine updates `media_items.relative_path` and refreshes `sidecar_assets` in the database.

## Audit

Every successful file rename produces an audit entry with action `file_rename`, the media item ID (or null for TV show batches), the new target path, and a detail string describing the old and new paths.

## Durable job integration

The rename job handler (`rename_execute`) is registered alongside the existing `scan` handler in the library service constructor. It deserializes the plan ID from the job payload, calls `ApplyRenamePlan` with the worker's progress callback, and reports per-file progress via the existing SSE event stream.

Cancellation is cooperative through the job context. Failed jobs can be retried through the existing job retry mechanism.

## UI

Both Movie File Audit (`MovieFileAuditTab.vue`) and TV File Audit (`TVFileAuditTab.vue`) feature:

- **Scope Selector** (for TV): Whole show, single season, or single episode.
- **Preset dropdown**: Kodi Standard, Kodi (with Title), Plex Standard, Jellyfin / Emby, Flat, and Custom patterns.
- Token pills for quick template insertion.
- **Dry Run Simulation** button generating real server-side rename plans.
- Semantic colored operation badges (`RENAME FILE`, `RENAME DIR`, `KEEP`, `CONFLICT`).
- **Sticky bottom action bar** with "Execute Safe Rename & Move" button when no conflicts exist.
- Full i18n support in English and Simplified Chinese.
