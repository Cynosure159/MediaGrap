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

The scanner accepts `mkv`, `mp4`, `m4v`, `avi`, `mov`, and `webm`, skips symbolic links, extracts basic title/year hints from filenames, and records matching NFO/image sidecars. It never changes media files.

