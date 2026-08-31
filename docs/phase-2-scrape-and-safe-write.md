# Phase 2 — Movie scrape and safe write

Phase 2 provides the first usable movie metadata workflow: locate a TMDb match, review and edit a draft, then explicitly preview and create a Kodi movie NFO.

## Included behavior

- TMDb movie search and details adapter behind a provider interface.
- An outbound HTTP client that honors `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, and `NO_PROXY`. `MEDIAGRAP_OUTBOUND_PROXY` explicitly overrides the proxy URL for MediaGrap requests. The present implementation supports HTTP/HTTPS proxy URLs; SOCKS support is intentionally deferred until it can share the same tested transport policy.
- Metadata drafts persisted in SQLite, including source/provider identifiers, titles, year, plot, runtime, genres, and selected image URLs.
- Deterministic Kodi movie NFO XML serialization and strict XML parsing.
- When an indexed movie has no saved MediaGrap draft, its same-basename Kodi NFO is read on opening the detail view and used as the editable starting draft. If it is absent, the Kodi-standard directory-level `movie.nfo` is used as a fallback. The original NFO remains unchanged until an explicit write plan is applied.
- An explicit write-plan preview. Apply refuses an existing destination, validates the configured media-root boundary, writes a temporary file in the target directory, synchronizes it, and atomically renames it into place.
- An audit record for preview and apply actions.
- A responsive movie detail panel, TMDb match selector, editable draft, NFO preview dialog, and a persisted English/Chinese UI choice.

## Configuration

| Variable | Purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | TMDb v3 API key. It is only read from the environment and is never returned from the API or logged. |
| `MEDIAGRAP_OUTBOUND_PROXY` | Optional HTTP/HTTPS proxy URL for provider requests. |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard proxy environment variables used when `MEDIAGRAP_OUTBOUND_PROXY` is absent. |

Add the TMDb key to the `environment` section of your private Compose override; do not commit it to this repository.

## Deliberate boundaries

Artwork download, Fanart.tv, provider caching/rate-limit persistence, write-plan cancellation/retry jobs, and existing-NFO merge/overwrite choices remain subsequent Phase 2 increments. This first slice never overwrites sidecars and does not download image files; it only stores selected artwork URLs in the draft and NFO preview. Existing NFOs can be read as the initial draft, but they are not automatically persisted or merged with a provider result.
