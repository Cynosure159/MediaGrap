# MediaGrap

**English** | [简体中文](README-zh.md)

A lightweight, self-hosted **movie and TV metadata manager** for NAS devices and home servers. Scan mounted libraries, match metadata, inspect files, and manage Kodi NFO and artwork from a desktop browser or mobile PWA. It is not a player, downloader, or transcoder.

## What works today

- Movie and show → season → episode catalogs; existing NFO and local artwork discovery.
- TMDb matching, metadata editing, optional Fanart.tv artwork, and cached ffprobe inspection.
- NFO/artwork write plans, movie/TV rename previews and background execution, jobs and audit history.
- Scheduled scans, provider proxies, Webhooks and source-scoped MCP automation with browser approval for movie file writes.
- English/Chinese UI, light/dark/system themes, responsive PWA; one non-root container, embedded Vue UI, Go server and SQLite WAL.

Early development continues; there is no automatic backup/restore or global file-operation rollback. See [usage and limitations](docs/usage.md) and the [roadmap](docs/roadmap.md).

## Try Docker

Use `cynosure159/mediagrap:latest` for a quick start, or follow [source builds](docs/distribution.md). Pin a reviewed version or image digest for production.

**First-run warning:** whoever reaches a fresh instance first can create its administrator. Complete setup on a trusted, isolated host/container network before allowing other users or a public proxy to connect. A loopback port mapping reduces host exposure but does not block direct container-network access; the application has no bootstrap token yet.

On a Linux Docker host, prepare only dedicated application state directories:

```sh
sudo install -d -m 0750 -o 65532 -g 65532 runtime/config runtime/cache
```

Replace `/srv/media` with your existing library directory. Start read-only while evaluating:

```sh
docker run -d --name mediagrap --restart unless-stopped \
  --security-opt no-new-privileges:true \
  -p 127.0.0.1:8080:8080 \
  -e MEDIAGRAP_CONFIG_DIR=/config \
  -e MEDIAGRAP_CACHE_DIR=/cache \
  -e MEDIAGRAP_MEDIA_ROOTS=/media \
  --mount type=bind,src="$(pwd)/runtime/config",dst=/config \
  --mount type=bind,src="$(pwd)/runtime/cache",dst=/cache \
  --mount type=bind,src=/srv/media,dst=/media,readonly \
  cynosure159/mediagrap:latest
```

Open <http://127.0.0.1:8080> on the Docker host and create your administrator. For a remote NAS, use a trusted SSH tunnel or another isolated access path. Do not run the app as root or recursively change media ownership. NAS ACLs/SELinux may need host-specific configuration; UID/GID **65532** needs state write access and media read/traverse access.

| Container path | Purpose |
| --- | --- |
| `/config` | Persistent database, settings, sessions, jobs and audit state; sensitive |
| `/cache` | Disposable cache/temporary data |
| `/media` | Explicitly mounted media; Settings source paths must be inside the startup allowlist |

For HTTPS termination, set `MEDIAGRAP_SECURE_SESSION_COOKIE=true` and restrict the backend to your trusted proxy. Do not enable it with browser-side plain HTTP. See [deployment, TLS, permissions and troubleshooting](docs/deployment.md).

## First library workflow

1. In **Settings → Media sources**, add the container path `/media` (not `/srv/media`). Mounting/allowlisting alone does not create a source.
2. Scan it. Scans update the database index, not media/NFO files, and do not automatically scrape.
3. Open **Movies** or **TV shows**, select an item, and inspect Overview, Artwork, Cast, NFO Raw and File Audit. Saved metadata takes priority over external NFO edits.
4. Configure your provider key in Settings when ready to match. **Selecting a TMDb candidate replaces metadata and immediately writes NFO; it is not a read-only preview.** TV matching can write multiple NFOs. Read-only evaluation must stop before this action.
5. To write, back up sidecars, recreate the container with the media mount writable, and grant only necessary filesystem permissions. Review manual NFO/artwork and rename plans before confirming; check Jobs/Audit afterward. No operation guarantees global rollback.

Automatic scan refreshes preserve in-progress edits; missing or uncertain targets disable new writes. Navigation/reload is not persistent draft storage. Details, rename tokens and integration setup: [usage](docs/usage.md), [Webhooks and MCP](docs/integrations.md).

## Upgrade and back up

Stop the container, back up the **entire config directory** plus any separately mounted Webhook key, and back up media-side NFO/artwork separately. Never copy only a live `mediagrap.db`: SQLite WAL may contain committed data. Retain the previous image digest and matching backup; database downgrades are not promised. Recreate using a reviewed pinned image/digest, verify readiness and sources, then scan. [Backup and upgrade procedure](docs/deployment.md#backup-and-upgrade).

## Develop and contribute

From a local checkout, with Go 1.26+, Node 24.15+, npm 11+ and Make:

```sh
npm --prefix web ci
go mod download
MEDIAGRAP_LISTEN=127.0.0.1:8080 npm run dev
```

Vite prints the browser URL and proxies to the Go API; state lives in ignored `.local/`. Use synthetic media for tests. Native development builds are not complete release distributions. [Development and contributing](docs/development.md) · [Architecture and UI conventions](docs/architecture.md) · [Security and reporting](docs/security.md).

## License and source

The current project uses [AGPL-3.0-only](LICENSE). Third-party works and provider content retain their own terms; this does not relicense all historic commits. Current distribution builds expose their matching source through **Download source / 下载源码** in the UI and unauthenticated `/source`; a native development build without that archive returns 503. See [source/build/release instructions](docs/distribution.md) and [upstream legal materials](docs/legal/README.md). No future public repository URL is assumed.
