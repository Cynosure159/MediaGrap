# Product scope

## Product statement

MediaGrap is a resource-efficient media metadata manager for users who keep movie, TV, concert, or music files on a NAS or home server. It brings the core workflow of desktop tools such as tinyMediaManager and MediaElch into a responsive, installable web application.

The product manages metadata beside the media library; it is not a streaming server, transcoder, downloader, or media player.

## Design goals

- Run comfortably on modest x86-64 and ARM64 home servers.
- Deploy as one container with a persistent configuration volume and one or more bind-mounted media roots.
- Make every destructive or structural file operation previewable and auditable.
- Use standards-compatible NFO and artwork layouts, starting with current Kodi conventions.
- Work well on desktop browsers and mobile screens; support PWA installation and reconnect gracefully.
- Provide a Chinese/English interface switch, with language preference persisted per user and a safe default before sign-in.
- Allow global and per-provider HTTP/HTTPS/SOCKS5 proxies without exposing credentials.
- Make providers replaceable and failures isolated.
- Remain useful without continuous internet access after metadata and artwork are cached.

## Non-goals for initial releases

- Video playback, streaming, transcoding, subtitle acquisition, torrent/usenet integration, or media downloading.
- Multi-node distributed processing.
- Arbitrary code plugins loaded into the server process.
- Full feature parity with mature desktop applications in the first release.
- Editing audio ID3 tags; music support will initially use sidecar metadata only.

## Users and workflows

### Administrator

1. Starts the container and completes first-run setup.
2. Adds host bind mounts as logical media sources.
3. Configures language, country, naming policy, scraper credentials, and optional proxy.
4. Reviews health, storage access, jobs, and audit history.

### Library curator

1. Scans a source and reviews newly discovered or changed items.
2. Resolves unmatched or ambiguous titles.
3. Compares metadata candidates and selects fields/artwork.
4. Previews NFO, artwork, and rename operations.
5. Applies changes, retries failures, or restores supported sidecar-file backups.

## Feature priority

### MVP: movies

- First-run admin account and session-based authentication.
- Media source configuration using paths already mounted into the container.
- Incremental and full scan with extension, ignore-pattern, symlink, and separate-folder policies.
- Filename parsing for title/year/edition and existing NFO discovery.
- TMDb search/details/images provider; optional Fanart.tv artwork provider.
- Candidate selection, manual metadata editing, field-level dirty state, and validation.
- Kodi movie NFO read/write, poster/fanart/logo/trailer sidecar discovery, and artwork download.
- Persistent background jobs with progress, cancel, retry, and restart recovery.
- File-operation dry run, conflict reporting, safe write, backup policy, and audit log.
- Responsive Web UI, dark/light themes, keyboard-friendly desktop use, installable PWA.
- Simplified Chinese and English UI, including setup, validation, job states, empty/error states, and file-operation warnings.
- Docker image for linux/amd64 and linux/arm64; health check and Compose example.
- HTTP, HTTPS, and SOCKS5 proxy configuration with no-proxy rules.

### v0.2: TV shows

- Show/season/episode discovery, episode filename parsing, and missing-episode reporting.
- Show and episode metadata scraping, season artwork, actor artwork, and Kodi NFO output.
- Batch operations with explicit scope and progress.

### v0.3: library operations

- Rename/move template engine with conditionals, sanitization, collision detection, and dry run.
- Duplicate detection and filters for missing metadata/artwork/IDs.
- CSV export and Kodi JSON-RPC synchronization.
- Webhook notifications and scheduled scans.

### Later

- Concert/music sidecar metadata, more providers, HTML export themes, multiple users/roles, PostgreSQL, external workers, and documented API tokens.

## Core domain terminology

- **Source**: a configured logical library root inside the container.
- **Media item**: movie, show, season, episode, concert, artist, or album.
- **Asset**: media file, NFO, poster, fanart, trailer, subtitle, or other sidecar.
- **Provider**: external metadata or artwork service.
- **Candidate**: a provider search result that may match a media item.
- **Change set**: a validated plan of filesystem/database changes awaiting application.
- **Job**: durable asynchronous work such as scan, scrape, download, or rename.

## Success measures

- Idle container memory target: under 100 MiB for a typical small library after stabilization; measure and publish rather than promise a hard limit prematurely.
- Cold image size target: under 100 MiB compressed, excluding user data.
- No unreviewed rename/move/delete operation from the Web UI.
- A stopped container can restart without losing job state or corrupting an NFO write.
- A 10,000-file library scan uses bounded memory and exposes progress.
- Mobile workflows work at 360 CSS pixels without horizontal page scrolling.

## MVP acceptance criteria

- A new user can deploy with Compose, mount a read/write movie directory, sign in, add the mounted path, scan, match a movie, review metadata, and write valid NFO/artwork.
- The same image runs on amd64 and arm64 as a non-root process.
- Scan, scrape, and artwork jobs survive application restart in a defined state.
- Invalid paths, symlink escapes, naming collisions, provider timeouts, rate limits, partial downloads, and insufficient permissions produce actionable errors.
- Proxy connectivity can be tested before saving and secrets are redacted from logs/API responses.
- PWA install metadata, responsive navigation, offline application shell, update prompt, and reconnect state are verified.
