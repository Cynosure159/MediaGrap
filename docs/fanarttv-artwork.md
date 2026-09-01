# Fanart.tv Artwork Workflow

MediaGrap's movie Artwork workshop follows the explicit-selection workflow used
by MediaElch: provider artwork is loaded asynchronously, grouped by Kodi
artwork type, sorted by the configured TMDb language. After a refresh, the
first candidate is preselected only for artwork types without a recognized
local file; existing artwork types remain unselected so they are not replaced
accidentally.

Supported movie groups are `poster`, `fanart`, `clearlogo`, `clearart`,
`discart`, `banner`, and `landscape`. Candidate records store only normalized
provider fields (source, preview, language, likes, dimensions, and MIME type).
Fanart.tv response payloads do not enter the domain model.

## Configuration

The Settings page accepts a Fanart.tv project API key. The key is persisted in
the SQLite application settings table and is never returned to the browser.
The environment equivalent is `MEDIAGRAP_FANARTTV_API_KEY`. The existing
outbound proxy and TMDb language setting are reused for Fanart.tv requests.
The browser never requests Fanart.tv directly: candidate previews are served
through `GET /api/v1/media/{id}/artwork-preview/{candidate}`. The server-side
preview request and the background artwork download share the same configured
outbound HTTP client, so both paths use the proxy and keep provider credentials
out of browser requests.

## Safe apply flow

1. `POST /api/v1/media/{id}/artwork-candidates` refreshes the candidate catalog
   from Fanart.tv's v3.2 movie endpoint using the matched TMDb ID.
2. `POST /api/v1/media/{id}/artwork-plans` accepts candidate IDs and creates a
   preview-only plan. The server rechecks ownership, provider host, artwork
   kind, target path, and conflicts; clients cannot submit arbitrary download
   URLs.
3. Applying a plan queues the durable `artwork_download` job. The worker
   downloads only HTTPS JPEG/PNG assets from the approved TMDb or Fanart.tv
   image hosts, validates MIME and magic bytes, stages beside the target, and
   atomically renames after a final path/conflict check.
4. The plan and job expose queued/running/succeeded/failed state. Artwork
   download jobs retry bounded failures and keep audit entries for preview and
   apply outcomes.

The current slice intentionally keeps custom upload and crop outside this
workflow; those remain a later feature because they need their own image
decoding, quota, and replacement policy.
