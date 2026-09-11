# Settings

The authenticated **Settings** page keeps operational configuration out of the media workspace.

## Provider settings

- A TMDb v3 API key can be supplied and changed at runtime. The API never returns the saved value; it only reports whether one is configured.
- An optional Fanart.tv project API key can be supplied and changed at runtime. It is used for movie and TV artwork candidates and is never returned; the API only reports whether one is configured.
- A separate optional Fanart.tv personal client key can be stored for earlier access to newly approved artwork. It is sent only from the server and is never returned or logged.
- TMDb search and detail requests use the selected information language. A separate fallback language fills missing translated title/overview fields on detail requests.
- An optional HTTP, HTTPS, `socks5://`, or `socks5h://` outbound proxy applies to provider and artwork requests immediately after saving. Proxy values and credentials are never returned.
- Comma-separated `NO_PROXY` host/domain/IP rules bypass the configured proxy. The API reports only whether rules exist.
- The same proxy and TMDb language preference apply to Fanart.tv artwork requests.
- Authenticated connection tests are limited to fixed TMDb/Fanart.tv endpoints. The server persists only target, result, HTTP status, duration, a redacted message, and timestamp; arbitrary test URLs are not accepted.

Settings are stored in the SQLite database on the `/config` volume. Protect that volume as application-sensitive data. Environment variables remain useful for first-run defaults and non-interactive deployment:

| Variable | Default / purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | First-run TMDb v3 API key fallback. |
| `MEDIAGRAP_FANARTTV_API_KEY` | First-run Fanart.tv project API key fallback. |
| `MEDIAGRAP_FANARTTV_PERSONAL_API_KEY` | Optional first-run Fanart.tv personal client-key fallback. |
| `MEDIAGRAP_TMDB_LANGUAGE` | First-run TMDb information language; defaults to `en-US`. |
| `MEDIAGRAP_FALLBACK_LANGUAGE` | First-run metadata fallback language; defaults to `en-US`. |
| `MEDIAGRAP_OUTBOUND_PROXY` | First-run HTTP/HTTPS/SOCKS5 outbound-proxy fallback. |
| `MEDIAGRAP_NO_PROXY` | First-run proxy bypass rules; falls back to standard `NO_PROXY`. |
| `MEDIAGRAP_FFPROBE_PATH` | Optional ffprobe executable; defaults to `ffprobe`. Set `off` to disable media stream probing. |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard environment proxy behavior when no MediaGrap proxy is configured. |

Once a value is saved in Settings it takes precedence over the corresponding environment default. Clearing a saved setting explicitly disables that setting for this instance.

## Media directories

The Settings page also manages indexed media sources. A source must be an existing directory beneath a startup media-root allowlist (`MEDIAGRAP_MEDIA_ROOTS`, default `/media`) and must already have been mounted into the container. The UI cannot browse or grant access to arbitrary host paths.

Configure one logical source at the common mounted parent directory, normally `/media`. For example, mounts such as `/srv/media/movies:/media/movies:rw` and `/srv/media/shows:/media/shows:rw` can be indexed together by adding `/media` once. The scanner derives TV shows from episode filenames and lists all other indexed video files as movies; sources do not have separate Movie or TV types. To permit NFO creation, mount the relevant child paths read-write, then rescan after changing filesystem contents.

Removing a media source removes only its MediaGrap configuration and indexed database records. It does not modify, move, or delete any mounted media files.

Each source has a persisted scan policy:

- `incremental` indexes new and changed files without marking unseen records missing;
- `full` first marks existing records missing, then reconciles everything found on disk;
- schedules use a 15-minute to 7-day interval, survive restarts in SQLite, avoid duplicate queued/running scans, and expose last/next run times in system status.

## Media inspection

Movie Overview and File Audit request media-stream information from the optional backend ffprobe adapter. A successful result is cached in SQLite until the media file size or modification time changes. The filesystem audit does not require ffprobe and continues to report real file size, MIME type, permissions, XML validity, read-only state, and symlink warnings when probing is unavailable.

The standard image bundles a statically linked, metadata-only ffprobe at `/usr/bin/ffprobe` and sets `MEDIAGRAP_FFPROBE_PATH` automatically. It probes local files only and covers the scanner's supported containers; unsupported formats are reported as unavailable. Set `MEDIAGRAP_FFPROBE_PATH=off` only when deliberately disabling probing. The UI never installs packages and never executes a host “open file/folder” command.

## Interface preferences

Display language and theme (`dark`, `light`, or `system`) are persisted per authenticated user and cached locally for immediate rendering. They do not affect TMDb data language; configure that separately in Provider settings.

## External integrations and file approvals

The settings page now includes Webhook endpoint management, scoped MCP API Tokens and administrator approval of automation movie file plans. Secrets are shown once after creation or rotation; they are not saved in browser storage. File approval displays the frozen relative paths, overwrite markers, NFO contents or artwork preview and requires explicit review before granting a five-minute approval. MCP cannot approve its own plans.

See [Webhook and MCP configuration](integrations.md) for key-file setup, network allowlists, token scopes, task tools and recovery limits.

The connection sidebar displays the latest test result (reachable, failed, or not configured) and duration for TMDb and the proxy. API Tokens distinguish active, expired, and revoked credentials; expiry status updates while the page remains open.

The Movies and TV shows catalog refresh buttons scan every enabled media source, including sources with no indexed items yet. Settings retains per-source scanning. Repeated catalog clicks during an active scan request do not queue duplicate scans.

## Default rename patterns

The separate **Rename Patterns** settings tab (`/settings?section=renaming`) stores movie and TV templates in the server database. `movieRenamePattern` and `tvRenamePattern` are optional fields on the settings update API: omitted fields keep their values; empty or malformed templates are rejected. The settings screen lists supported tokens and provides reset buttons; reset changes must be saved.

New file-audit tabs load the saved template for their media type. In-progress edits and existing immutable plans are not replaced. Saving defaults never writes media; preview, conflict checks and explicit execution remain required. `/` separates relative directories, extensions are preserved, and absolute paths, traversal and unknown tokens are rejected. Metadata-dependent tokens still require corresponding values during preview.
