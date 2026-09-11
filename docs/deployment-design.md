# Deployment design

## Production shape

The standard distribution is a multi-architecture OCI image containing one non-root Go process and embedded frontend assets. Node.js and build tools exist only in builder stages.

Recommended container contract:

| Container path | Purpose | Mode |
| --- | --- | --- |
| `/config` | SQLite database, configuration, keys, audit/job state | read/write |
| `/cache` | provider responses, temporary artwork, thumbnails | read/write; disposable |
| `/media/...` | one or more user-defined library mounts | read-only for evaluation, read/write for NFO/artwork/rename |

Do not expose arbitrary host browsing in the UI. An administrator mounts host paths explicitly and then selects only container-visible paths under an allowlisted prefix.

## Compose target

The implementation should provide a Compose example equivalent to:

```yaml
services:
  mediagrap:
    image: ghcr.io/OWNER/mediagrap:VERSION
    container_name: mediagrap
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      TZ: Asia/Shanghai
      MEDIAGRAP_CONFIG_DIR: /config
      MEDIAGRAP_CACHE_DIR: /cache
      MEDIAGRAP_FFPROBE_PATH: /usr/bin/ffprobe
    volumes:
      - ./config:/config
      - ./cache:/cache
      - /srv/media/movies:/media/movies
      - /srv/media/tv:/media/tv
    healthcheck:
      test: ["CMD", "/mediagrap", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
    security_opt:
      - no-new-privileges:true
```

The exact image name and version are placeholders until publishing is configured. Production documentation should recommend a pinned version, not `latest`.

The `healthcheck` command probes `/readyz` on the running process rather than opening a separate SQLite pool. Use `MEDIAGRAP_LISTEN` for a custom container listener; host port mappings do not change that listener. See [database diagnostics](database-diagnostics.md) for short-timeout probes and pool-exhaustion troubleshooting.

## Permissions

- Run as a non-root numeric UID/GID and document ownership requirements.
- Prefer Compose `user: "UID:GID"` over an entrypoint that starts as root and drops privileges.
- On platforms where environment-based `PUID`/`PGID` compatibility is necessary, implement it explicitly and document the security tradeoff.
- Perform a source access check that distinguishes read, create, rename, and delete permissions before enabling writes.
- Support read-only mounts for scan/scrape review; disable apply actions with a clear reason.

## Configuration precedence

1. Command-line flags.
2. `MEDIAGRAP_*` environment variables.
3. Persisted non-secret settings in SQLite.
4. Safe built-in defaults.

Environment variables are appropriate for bootstrap values and secret injection. The Web UI may manage operational settings after authentication. Secret responses indicate only whether a value is configured; they never return the plaintext value.

## Proxy support

Support these inputs:

- standard `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, and `NO_PROXY` environment variables;
- global proxy configured in the Web UI;
- optional per-provider override;
- schemes `http`, `https`, and `socks5`/`socks5h`;
- proxy authentication stored encrypted-at-rest when a stable application key is provided.

All outbound metadata, image, update-check, and provider-test requests use the same transport policy unless explicitly exempted. The settings UI provides a redacted summary and a connectivity test to a known provider endpoint. Logs strip URL user info and authorization headers.

## TLS and network exposure

The default server listens on port 8080. For LAN-only use it can be published directly; internet exposure should sit behind a trusted reverse proxy providing TLS. Respect forwarded headers only from configured trusted proxies.

First-run setup is security-sensitive. The implementation should either require a bootstrap token printed once to logs or restrict setup to a clearly configured trusted network. Binding publicly with an unauthenticated setup page is not acceptable.

## Backup and restore

- A consistent backup includes `/config`; use an application backup command or SQLite online backup API rather than copying an active database blindly.
- `/cache` need not be backed up.
- Media-side NFO and artwork are already part of the media storage backup policy.
- Schema migration creates a pre-migration database backup when practical and refuses to start if available disk space is clearly insufficient.
- Document downgrade compatibility per release; never imply all schema downgrades are supported.

## Updates and image publishing

- Build linux/amd64 and linux/arm64 images.
- Publish immutable semantic-version and digest-addressable tags.
- Generate an SBOM, scan dependencies/images, and sign release images when CI is established.
- Use the repository's `ffprobe` target as the standard production image, published locally as `latest`: `docker build -t mediagrap:latest .`. The build compiles a small, statically linked ffprobe with the media containers and codecs currently needed by inspection, then copies only that binary and CA certificates into a scratch runtime. It accepts local files only and covers the scanner's `avi`, `flac`, `matroska`, `mp3`, `mpegts`, `mov`, `ogg`, and `wav` containers; unsupported formats remain an explicit unavailable probe state. The FFmpeg source version is pinned by the `FFMPEG_VERSION` build argument (currently `7.1.1`).
- Filesystem audit remains available independently of probing. Set `MEDIAGRAP_FFPROBE_PATH=off` only for an intentional no-probe deployment.

## Local development

- Go server and Vite dev server run separately with API proxying and hot reload.
- A production-like local command builds frontend assets, embeds them, migrates a temporary SQLite database, and starts the single binary.
- Integration tests use temporary directories as media roots; they must never point at a developer's real media library.

## Webhook signing and MCP

Webhook delivery requires a separately backed-up, mode-0600 master key file containing a 32-byte key encoded as hex. Mount it read-only and set `MEDIAGRAP_WEBHOOK_KEY_FILE`; the process will not replace a missing key. Webhook network policy is configured by deployment environment, independently of provider proxies. Expose MCP `/mcp` through TLS and restrict accepted browser origins; static API Tokens are managed through the authenticated settings page.

See [integration deployment variables and limits](integrations.md) for the complete configuration. The feature adds no container or external worker. Browser UI approval is required before an MCP movie file plan may execute.

### Native development startup

From the repository root, run `npm run dev` (or `npm start -- dev`) to start the Go API and Vite together. Install frontend dependencies first with `npm --prefix web install`. Running `npm --prefix web run dev` alone starts only Vite; media access is handled by the Go API.

The Makefile reads the ignored `.env.local` file for local media allowlists:

```dotenv
MEDIAGRAP_MEDIA_ROOTS=/path/to/media/movies,/path/to/media/shows
```

Add those absolute paths as sources in Settings once, then scan them. Source configuration persists in `.local/config`; caches use `.local/cache`. Changing the allowlist does not automatically create sources or modify media files. Restart the development services after changing `.env.local`.
