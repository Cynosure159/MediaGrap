# Library usage and settings

[Deploy first](deployment.md) · [中文入口](../README-zh.md#第一个媒体库) · [Automation](integrations.md)

This is the current-source behavior guide, not a feature matrix for the older 0.0.6 image. The UI's English and Chinese labels refer to the same operations. MediaGrap manages metadata and sidecars, not playback, downloads, transcoding or audio tag editing.

## Sources and scanning

Add an existing **container-visible** source in Settings beneath the startup `MEDIAGRAP_MEDIA_ROOTS` allowlist (default `/media`). For mounts `/media/movies` and `/media/shows`, a single `/media` source normally suffices; sources have no separate Movie/TV type. UI source selection cannot grant host access. Removing a source removes its configuration and indexed records, never its media files.

Scans are read-only for media/NFO/artwork, write only the SQLite index and eligible local metadata, and do not call providers automatically. Accepted video extensions are `mkv`, `mp4`, `m4v`, `avi`, `mov`, `webm`; symlinks are skipped. TV examples: `Show.S02E03.mkv`, `Show - 1x02 - Pilot.mp4`, `Show.S01E01E02.mkv`. Conventional `Show/Season 01/` directories supply show hints; Season 0 is supported. Episode-named files in `Show/Specials/` (case-insensitive exact folder name) also belong to `Show`, even without regular seasons. The folder supplies show identity, not season/episode numbers: filename tokens such as `S00E01` remain authoritative for scan classification, and existing NFO/saved metadata is not rewritten. A source-relative top-level `Specials/` is not folded into the source root, so a literal show named `Specials` keeps its identity; sources pointing directly at a show directory retain this conservative boundary. Other indexed videos appear as movies. One multi-episode file currently uses the first recognized episode for display/NFO; this is not full multi-episode NFO or official missing-episode reporting.

Sources support manual `incremental`/`full` scans and persisted interval schedules from **15 minutes to 7 days**. Catalog refresh scans all enabled sources, including empty ones; Settings permits per-source scans. Queue insertion suppresses another queued/running `scan` for the same source (HTTP 409 on duplicates); retries check this too. This is not a guarantee covering every automation wrapper kind. There is one media-job worker; cancelled handlers can still be finishing cooperative cleanup.

- **Incremental** still traverses the entire tree, but skips unchanged index rewrites using stat-based video/sidecar fingerprints. It is not a filesystem watcher.
- **Full** bypasses fingerprints and revalidates index classification; it does not force local NFO edits over saved metadata. Run a full source scan to repair previously indexed separate `Show/Specials` show entries; an incremental scan can retain their old fingerprinted grouping. Saved metadata on the old show is retained, not automatically merged into the parent show.
- **Both modes** reconcile unseen files as missing **only after successful complete traversal and final commit**, together with `last_scan_at`. Deleted movies/episodes and empty shows then disappear from catalogs; metadata remains stored. Failed/cancelled scans retain earlier committed batches without declaring unseen files missing.
- An accessible empty mountpoint is treated as an empty source. Check disconnected NAS mounts before scanning; the app cannot infer mount intent.
- Same-size replacements preserving mtime/mode can evade incremental detection. Another scan may be needed for files changed during enumeration. No whole-filesystem atomic snapshot is promised.

Sample clips with final `-sample`, `.sample`, `_sample` tokens or standalone `sample` basename are excluded case-insensitively; `The Sample (2024)` is not. Exact `Extra`/`Extras` child directories are skipped only beneath a current or historically indexed direct main video, not merely because a source/show has that name. Old supplemental records are retired only on successful final reconciliation, without file deletion. There is no separate extras collection yet.

## Existing metadata and catalog

**Saved SQLite metadata wins over local NFO.** A movie without a saved title can import its same-basename `.nfo`, falling back to Kodi `movie.nfo`. Saved provider/locks are preserved even if a scan's NFO parse races a manual save. `provider=nfo` does not authorize future overwrites: a previously imported record may have been edited manually. NFO changes/deletion refresh sidecar discovery, not saved metadata. Missing metadata with an unchanged NFO is retried after an interrupted/failed import. Empty, malformed, oversized, non-regular or unreadable NFO gives a non-blocking `invalid_nfo` warning and filename hints; it is never silently rewritten. Database failures remain errors.

Local image paths are indexed as sidecars; supported discovery formats include JPEG, PNG and WebP. A single-main-video directory attributes directory-level images as well as same-basename files. Local images use authenticated restricted asset routes, not host paths. TV details can import `tvshow.nfo` and episode NFO when saved records are absent; saved TV titles take precedence over filename hints. Show `poster`/`folder`/`cover` and `fanart`/`backdrop`/`landscape` images take priority for display.

Movie catalogs fetch **50 records per request**, load on approach to the viewport bottom, and show true server totals. Global search/filter/sort operates on the server. Available API filters are `all`, `unscraped`/`nfo_missing`, `poster_missing`, `4k`, `1080p`; sorts are `title`, `year`, `size`, with stable ID tie-breakers. The current UI exposes All, Unscraped, 4K and 1080p. NFO/image filters use the index; resolution filters use filename hints, not ffprobe. Year display/sort prefers saved metadata then the filename year hint. Rendered rows are virtualized; loaded lightweight records remain in memory. Refresh after concurrent changes for a consistent new ordering.

## Inspect, match and write

Select an item and use **Overview, Artwork, Cast, NFO Raw, File Audit**. TV selection scopes the inspector to show, season or episode; cast is a show-level field, not invented per-season credits.

**Searching is not the same as selecting a match. Selecting a TMDb candidate is a replacement action: it persists metadata and immediately writes NFO.** It is not an editable draft preview and may replace an existing regular NFO. Movie matching defaults to parent-directory title/year where available; root-level files fall back to filename hints. Review the candidate and backup policy before selecting; never use this action as a read-only test. Path/type/symlink checks, temporary-file publication and audit still apply.

Movie details include title/original title, year, plot, runtime, genres, rating/votes, certification (preferred language region then US), studios, directors, writers and up to 20 cast members/roles/portrait URLs. Directors/writers/studios can be edited; cast comes from the match. Portrait URLs are not a batch of downloaded local headshots.

Manual **Save & Write NFO** saves metadata and follows the NFO plan/apply workflow; inspect the generated XML/overwrite target when offered. Metadata persistence and filesystem publication are separate: if availability cannot be verified after metadata save, the UI explicitly reports **metadata saved, NFO not written**, retains input and blocks further writes until rechecked. Never assume a failed NFO write rolled back saved metadata.

TV show matching fetches only locally discovered seasons, replaces the show's saved metadata and writes `tvshow.nfo`, discovered `season.nfo` files and same-basename episode NFOs. After the show has a TMDb ID, season scraping replaces only that season's episode metadata/NFO targets; episode scraping replaces only that episode. These browser flows can write NFO immediately, unlike the database-only MCP TV tools. Local images and video contents are not changed by matching. Maintenance NFO plan endpoints also permit individual review/apply. Unknown XML extensions are not preserved by generated replacement NFOs.

Manual NFO/artwork/rename previews show scope and intended changes. Execute only after explicit review, on writable mounts; apply rechecks paths and conflicts. Existing symlink/non-regular sidecar targets are blocked. Same-directory temporary files and atomic replacement are used where supported; **no durable sidecar backups or cross-file global rollback exist**. The NFO Raw tab is not a promise of a free-form XML editor or alternate Plex/Jellyfin serialization.

## Artwork

Movie artwork groups: `poster`, `fanart`, `clearlogo`, `clearart`, `discart`, `banner`, `landscape`. Fanart.tv movie lookup needs a positive numeric TMDb ID, from a match or explicitly typed NFO `uniqueid type="tmdb"`; IMDb IDs are not interchangeable. An NFO-imported movie need not be rescraped just to get Fanart.tv candidates.

TV Fanart.tv lookup uses TheTVDB ID, obtained through TMDb external IDs or local `uniqueid type="tvdb"`/legacy `tvdbid`. Show artwork includes poster, background, logo/clearlogo, clearart, banner, landscape and character art. Seasonal candidates retain their numeric season and write `seasonNN-poster`, `seasonNN-banner`, `seasonNN-landscape` in the show directory. Episodes use TMDb still/local same-basename images rather than this Fanart.tv seasonal set.

Candidates rank preferred language, neutral, English fallback, other languages, then likes. Refresh preselects only kinds with no recognized local image; existing kinds stay unselected. Server-side previews use the shared provider transport, keeping keys out of browser requests. Choose candidate IDs, inspect the write plan, then apply to queue `artwork_download`. Arbitrary client download URLs are not accepted. HTTPS image-host restrictions, redirects policy, JPEG/PNG MIME and magic checks, **25 MiB per response**, same-directory staging, final path/conflict checks and audit protect downloads. MIME determines `.jpg`/`.png`; there is no silent transcoding. Upload/crop and local portrait downloads are deferred.

## File audit and rename

File Audit reports source-relative paths, actual Lstat size/mtime/mode, MIME/XML checks and symlink/read-only warnings, independently of ffprobe. Optional ffprobe reads video/audio/subtitle/HDR details directly without a shell, with **30-second timeout and 8 MiB output cap**. Only successful results are cached by media ID, size and nanosecond mtime; fingerprint changes during probing prevent stale caching. Missing/disabled/unsupported probing is shown as unavailable, not synthetic stream data. No browser action runs a host “open folder” command.

Movie and TV File Audit provide saved template defaults, presets and persisted rename plans. TV scope can be whole show, season or episode. `/` creates relative directory structure; absolute paths, traversal, unknown tokens and empty templates are rejected. Extensions and sidecar language/edition suffixes are preserved. Inspect directory operations carefully: browser directory moves and automation per-file moves have different recovery boundaries.

Rename templates are 1–1024 UTF-8 bytes and use strict `${token}` expressions. A missing value is an error. Optional expressions use exactly `${prefix,token,suffix}`: they render `prefix + value + suffix` when the value is present, or nothing when it is missing. Whitespace around the token name is trimmed, while whitespace in affixes is retained. Empty or whitespace-only values are missing; the string `0` is present. Examples include `${,edition,}`, `${ - ,edition,}`, `${ (,year,)}`, and `${ [,resolution,]}`. `edition` and `imdbId` remain currently unavailable; optional use is therefore the safe form for those fields.

Unknown tokens, unsupported tokens, malformed expressions and expressions with extra commas fail even when optional. Nesting, scripting, arbitrary defaults, escaping, multi-token groups and full TMM compatibility are not supported. Affixes cannot contain separators, traversal, unsafe filename characters or control characters. Only literal `/` outside expressions creates directories; metadata is sanitized as filename text and can never create directory hierarchy. After optional omission and sanitization, empty filenames and empty, `.` or `..` directory segments are rejected. The legacy `naming-preview` endpoint remains filename-only and keeps its older supported-token set. Saving a template explicitly updates the persisted default; there is no automatic migration, and already-created plans remain unchanged and frozen. Plans require explicit preview/apply, conflict checks and audit.

| Template | Tokens |
| --- | --- |
| Movie | `${title}`, `${originalTitle}`, `${year}`, `${resolution}`, `${videoCodec}`, `${audioCodec}`, `${edition}`, `${imdbId}` |
| TV | `${showTitle}`, `${seasonNumber}`, `${seasonNumberPad}`, `${episodeNumber}`, `${episodeNumberPad}`, `${episodeTitle}`, `${year}`, `${resolution}`, `${videoCodec}`, `${audioCodec}` |

Example: `${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad}`. Values need matching metadata/probe fields. Presets named for media centers specify naming layouts, not alternate NFO compatibility guarantees. The older `naming-preview` API is an in-memory filename-only dry run, distinct from persisted `rename-plans`.

Settings → Rename Patterns (`/settings?section=renaming`) persists `movieRenamePattern` and `tvRenamePattern`. Omitted update fields retain values; empty/malformed values are rejected. Reset affects only the form until saved. New workshop instances load defaults; existing edits/plans do not change. Saving a template never writes media.

Plans report existing destinations and case-folded collisions; conflicting plans cannot be applied. Applying queues `rename_execute` with per-item progress/audit and possible partial/failed/cancelled outcomes. Browser same-device moves use rename; cross-device file moves use staged copy and size verification before source removal. This is not the automation engine's hash-verified recovery protocol. Check actual files and results before retrying partial moves. Successful video path changes clear scan fingerprints so the next incremental scan can reclassify movie/TV identity. [Automation movie plan limits](integrations.md#file-plans-and-approval) are separate.

## Jobs and drafts

Jobs persist in SQLite with `queued`, `running`, `succeeded`, `failed`, `cancelled`, `interrupted` states, progress and retry counters. Startup marks abandoned running work interrupted, not silently resumed file mutations. Cancellation is cooperative; completed file operations are not undone. Failed/cancelled/interrupted jobs may be retried where permitted; automation file work requires fresh reviewed plans instead of generic retry.

Jobs uses persisted SSE events with monotonic IDs and reconnection plus periodic snapshots. Default history is the latest 100 jobs; active scans have separate 100-row keyset pages and exact-ID lookups. Library tracking survives navigation/reload without a 60-second expiry. It refreshes every 2 seconds when active, 10 seconds idle, backs errors off to 30 seconds and bounds request/lookups. A short scan seen only in recent terminal history still refreshes once, including same-state retries. A completely unobserved burst that pushes a short job out of the latest-100 window is outside this bounded guarantee.

Automatic scan notifications do **not** remount inspectors or discard drafts. Edit mode, changed inputs, pending saves/dialogs and artwork/file workshops defer replacement, while read-only availability checks continue. Missing movie/show/episode targets retain dirty input but disable new writes, including apply dialogs. Network/5xx uncertainty is not proof of deletion, but also blocks writes until Retry/subsequent verification. Reappearance enables actions without overwriting dirty input; clean missing selections clear their detail. Generations reject late responses from previous selections. Explicit navigation/reload is not persistent draft storage; already-issued mutations are not blindly cancelled or duplicated.

Operations includes recent audit entries (up to 200 for NFO/artwork preview/apply), mount/cache/database state and redacted provider tests. `atomic_replace_only` means no old-file restore guarantee. See [operations troubleshooting](deployment.md#health-and-troubleshooting).

## Provider and interface settings

Authenticated Settings stores operational values in SQLite. Saved values override environment fallbacks; explicitly clearing a saved setting disables it rather than exposing the environment value again. Secrets/proxy values are write-only from API reads, not a claim of encryption at rest. Keep `/config` private. Fixed-target TMDb/Fanart.tv/proxy tests persist only target/result/status/duration/redacted message/time, never accept arbitrary test URLs.

| Environment fallback | Default / purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | TMDb v3 API key |
| `MEDIAGRAP_FANARTTV_API_KEY` | Optional Fanart.tv project key |
| `MEDIAGRAP_FANARTTV_PERSONAL_API_KEY` | Optional personal client key; earlier access to newly approved artwork |
| `MEDIAGRAP_TMDB_LANGUAGE` | `en-US`; search/detail information language and artwork preference |
| `MEDIAGRAP_FALLBACK_LANGUAGE` | `en-US`; fills missing translated detail title/overview |
| `MEDIAGRAP_OUTBOUND_PROXY` | Shared HTTP/HTTPS/`socks5`/`socks5h` proxy for provider/artwork traffic |
| `MEDIAGRAP_NO_PROXY` | Comma-separated host/domain/IP bypass rules; falls back to `NO_PROXY` |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard transport behavior when no MediaGrap proxy is configured |

There is one shared provider/artwork proxy, not implemented per-provider overrides. Webhook network policy is independent. UI language and `dark`/`light`/`system` theme persist per user and are locally cached; UI language is separate from TMDb data language. API Tokens, Webhooks and browser file approvals are documented in [integrations](integrations.md).
