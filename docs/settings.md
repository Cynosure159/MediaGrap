# Settings

The authenticated **Settings** page keeps operational configuration out of the media workspace.

## Provider settings

- A TMDb v3 API key can be supplied and changed at runtime. The API never returns the saved value; it only reports whether one is configured.
- An optional Fanart.tv project API key can be supplied and changed at runtime. It is used for movie artwork candidates and is never returned; the API only reports whether one is configured.
- TMDb search and detail requests use the selected information language. The first choices include `zh-CN`, `zh-TW`, `en-US`, `ja-JP`, and `ko-KR`; the API accepts any valid TMDb-style language code such as `de-DE`.
- An optional HTTP/HTTPS outbound proxy applies to TMDb requests immediately after saving. Proxy values are not returned in the API response.
- The same proxy and TMDb language preference apply to Fanart.tv artwork requests.

Settings are stored in the SQLite database on the `/config` volume. Protect that volume as application-sensitive data. Environment variables remain useful for first-run defaults and non-interactive deployment:

| Variable | Default / purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | First-run TMDb v3 API key fallback. |
| `MEDIAGRAP_FANARTTV_API_KEY` | First-run Fanart.tv project API key fallback. |
| `MEDIAGRAP_TMDB_LANGUAGE` | First-run TMDb information language; defaults to `en-US`. |
| `MEDIAGRAP_OUTBOUND_PROXY` | First-run HTTP/HTTPS outbound-proxy fallback. |
| `MEDIAGRAP_FFPROBE_PATH` | Optional ffprobe executable; defaults to `ffprobe`. Set `off` to disable media stream probing. |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard environment proxy behavior when no MediaGrap proxy is configured. |

Once a value is saved in Settings it takes precedence over the corresponding environment default. Clearing a saved setting explicitly disables that setting for this instance.

## Media directories

The Settings page also manages indexed media sources. A source must be an existing directory beneath a startup media-root allowlist (`MEDIAGRAP_MEDIA_ROOTS`, default `/media`) and must already have been mounted into the container. The UI cannot browse or grant access to arbitrary host paths.

Configure one logical source at the common mounted parent directory, normally `/media`. For example, mounts such as `/srv/media/movies:/media/movies:rw` and `/srv/media/shows:/media/shows:rw` can be indexed together by adding `/media` once. The scanner derives TV shows from episode filenames and lists all other indexed video files as movies; sources do not have separate Movie or TV types. To permit NFO creation, mount the relevant child paths read-write, then rescan after changing filesystem contents.

Removing a media source removes only its MediaGrap configuration and indexed database records. It does not modify, move, or delete any mounted media files.

## Media inspection

Movie Overview and File Audit request media-stream information from the optional backend ffprobe adapter. A successful result is cached in SQLite until the media file size or modification time changes. The filesystem audit does not require ffprobe and continues to report real file size, MIME type, permissions, XML validity, read-only state, and symlink warnings when probing is unavailable.

The standard image bundles a statically linked, metadata-only ffprobe at `/usr/bin/ffprobe` and sets `MEDIAGRAP_FFPROBE_PATH` automatically. It probes local files only and covers the scanner's supported containers; unsupported formats are reported as unavailable. Set `MEDIAGRAP_FFPROBE_PATH=off` only when deliberately disabling probing. The UI never installs packages and never executes a host “open file/folder” command.

## Interface language

The display-language selection is stored locally in the current browser/PWA. It changes the English/Chinese interface immediately and does not affect TMDb data language; configure that separately in Provider settings.
