# Phase 2 — Movie scrape and safe write

Phase 2 provides the first usable movie metadata workflow: locate a TMDb match, review and edit a draft, then explicitly preview and create a Kodi movie NFO.

## Included behavior

- TMDb movie search and details adapter behind a provider interface.
- Default TMDb search terms prioritize the indexed video's parent directory name (for example, `奥本海默 (2023)` becomes `奥本海默` with year `2023`). This avoids sending codec, resolution, and release-group tags from filenames; files directly in a source root fall back to their file-derived title.
- An outbound HTTP client that honors `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, and `NO_PROXY`. `MEDIAGRAP_OUTBOUND_PROXY` explicitly overrides the proxy URL for MediaGrap requests. The present implementation supports HTTP/HTTPS proxy URLs; SOCKS support is intentionally deferred until it can share the same tested transport policy.
- TMDb search and detail requests emit structured lifecycle logs. They record the safe request endpoint, status, duration, media-item ID, and candidate count; credentials, proxy values, and request URLs are never logged. Failures additionally distinguish timeout from other transport failures.
- Metadata drafts persisted in SQLite, including source/provider identifiers, titles, year, plot, runtime, genres, and selected image URLs.
- Deterministic Kodi movie NFO XML serialization and strict XML parsing.
- When an indexed movie has no saved MediaGrap draft, its same-basename Kodi NFO is read on opening the detail view and used as the editable starting draft. If it is absent, the Kodi-standard directory-level `movie.nfo` is used as a fallback. The original NFO remains unchanged until an explicit write plan is applied.
- An explicit write-plan preview. Saving replaces an existing same-basename NFO (or the directory-level `movie.nfo` fallback) only after the user confirms the preview. It validates the configured media-root boundary, writes a temporary file in the target directory, synchronizes it, and atomically renames it into place. Symlinks and non-regular targets remain blocked.
- A separate artwork download preview for the selected TMDb poster and fanart. It writes Kodi-standard `poster.jpg` and `fanart.jpg` next to the movie only after confirmation. Existing regular files may be atomically replaced after being shown in the preview; symlinks and non-regular targets are blocked.
- Artwork requests are limited to HTTPS `image.tmdb.org/t/p/…` URLs, do not follow redirects, require `image/jpeg`, and cap each response at 25 MiB. Downloads use the configured outbound proxy and stage every file in the target directory before replacement.
- An audit record for preview and apply actions.
- A responsive movie detail panel, TMDb match selector, editable draft, NFO preview dialog, and a persisted English/Chinese UI choice.
- On desktop, the movie list and active metadata inspector are a fixed master/detail workspace. On mobile, selecting a movie opens the inspector as the single visible pane with an explicit return-to-list action.

## Configuration

Provider, proxy, media-source, and interface options are now available from the authenticated Settings page. See [Settings](settings.md) for the persistence and security model. The TMDb information language saved there is used for both movie search results and selected-movie details.

| Variable | Purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | TMDb v3 API key. It is only read from the environment and is never returned from the API or logged. |
| `MEDIAGRAP_OUTBOUND_PROXY` | Optional HTTP/HTTPS proxy URL for provider requests. |
| `MEDIAGRAP_TMDB_LANGUAGE` | Default TMDb result/detail language (`en-US` by default). |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard proxy environment variables used when `MEDIAGRAP_OUTBOUND_PROXY` is absent. |

Add the TMDb key to the `environment` section of your private Compose override; do not commit it to this repository.

## Deliberate boundaries

Fanart.tv, provider caching/rate-limit persistence, durable download jobs with cancellation/retry, image format conversion, and field-level NFO merge choices remain subsequent Phase 2 increments. The current artwork slice downloads only the selected TMDb JPEG poster/fanart synchronously after an explicit preview. Existing NFOs can be read as the initial draft, but they are not automatically persisted or merged with a provider result.
