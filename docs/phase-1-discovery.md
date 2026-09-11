# Phase 1: read-only movie discovery

Phase 1 adds the first operational media-library slice. It is intentionally read-only with respect to user media: scans write only MediaGrap's SQLite index and never modify, rename, move, scrape, or download beside mounted media files.

## Configuration

`MEDIAGRAP_MEDIA_ROOTS` is a comma-separated allowlist of container-visible parent directories. It defaults to `/media`. A source can only be added when its normalized path is inside one of these roots and is an accessible directory.

## APIs

- `GET /api/v1/setup/status`, `POST /api/v1/setup`
- `POST|GET|DELETE /api/v1/session`
- `GET|POST /api/v1/sources`
- `POST /api/v1/sources/{id}/scans`
- `GET /api/v1/media?page=&pageSize=&q=`
- `GET /api/v1/jobs`

Browser mutations require an authenticated session cookie and `X-CSRF-Token`; first setup and login are explicit bootstrap operations.

## Scanner behavior

The scanner accepts `mkv`, `mp4`, `m4v`, `avi`, `mov`, and `webm`, skips symbolic links, extracts basic title/year hints from filenames, and records matching NFO/image sidecars. For a one-video movie directory, every supported local image (`jpg`, `jpeg`, `png`, or `webp`) is indexed, not only poster/fanart names. For a movie with no existing SQLite metadata, it parses a local Kodi NFO and persists the parsed fields in SQLite; discovered image paths remain in the SQLite sidecar index and are served through an authenticated opaque-asset URL. This rebuilds list and detail data after a source is removed and added again, without changing media files.

## Movie catalog browsing

The movie catalog now uses [continuous scrolling and lazy rendering](catalog-scrolling.md) with true server totals, global filters/sorts, and bounded background batches rather than stopping at the first 50 items. There are no page-number controls.

## Unreadable movie NFO fallback

`GET /api/v1/media/{id}` treats a rejected local NFO (empty, malformed, missing title, oversized, non-regular, or unreadable) as a non-blocking metadata warning when no saved metadata is available. It returns HTTP 200 with the selected item's indexed identity, filename title/year hints, local artwork, and `metadataWarning: "invalid_nfo"`. The UI shows a localized warning instead of a fatal parser error. Valid NFOs still hydrate metadata; existing saved metadata takes precedence. Database failures remain errors, not successful fallback responses. No rejected NFO is rewritten and fallback metadata is not persisted.

The movie inspector clears previous selection details, drafts and dialogs before fetching another movie. Detail requests are aborted and sequence-checked; selection-scoped inspector instances also prevent old child workshop state from carrying into another movie. A failed request cannot display the previous movie or enable editing its stale data under the new ID.

## Deferred UI localization requirement

The product now requires Simplified Chinese and English UI switching. The next UI iteration must use locale message keys rather than in-component user-facing text, select browser language before sign-in, persist a signed-in user's preference, and localize validation, task, safety, and error states as well as navigation labels. Provider metadata remains in its source language unless a later scraping setting requests a specific metadata locale.
