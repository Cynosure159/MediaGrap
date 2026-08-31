# Settings

The authenticated **Settings** page keeps operational configuration out of the media workspace.

## Provider settings

- A TMDb v3 API key can be supplied and changed at runtime. The API never returns the saved value; it only reports whether one is configured.
- TMDb search and detail requests use the selected information language. The first choices include `zh-CN`, `zh-TW`, `en-US`, `ja-JP`, and `ko-KR`; the API accepts any valid TMDb-style language code such as `de-DE`.
- An optional HTTP/HTTPS outbound proxy applies to TMDb requests immediately after saving. Proxy values are not returned in the API response.

Settings are stored in the SQLite database on the `/config` volume. Protect that volume as application-sensitive data. Environment variables remain useful for first-run defaults and non-interactive deployment:

| Variable | Default / purpose |
| --- | --- |
| `MEDIAGRAP_TMDB_API_KEY` | First-run TMDb v3 API key fallback. |
| `MEDIAGRAP_TMDB_LANGUAGE` | First-run TMDb information language; defaults to `en-US`. |
| `MEDIAGRAP_OUTBOUND_PROXY` | First-run HTTP/HTTPS outbound-proxy fallback. |
| `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY` | Standard environment proxy behavior when no MediaGrap proxy is configured. |

Once a value is saved in Settings it takes precedence over the corresponding environment default. Clearing a saved setting explicitly disables that setting for this instance.

## Media directories

The Settings page also manages indexed media sources. A source must be an existing directory beneath a startup media-root allowlist (`MEDIAGRAP_MEDIA_ROOTS`, default `/media`) and must already have been mounted into the container. The UI cannot browse or grant access to arbitrary host paths.

For example, a Compose mount of `/srv/media/movies:/media/movies:ro` allows `/media/movies` to be added as a source. To permit NFO creation, mount that particular path read-write, then rescan after changing filesystem contents.

## Interface language

The display-language selection is stored locally in the current browser/PWA. It changes the English/Chinese interface immediately and does not affect TMDb data language; configure that separately in Provider settings.
