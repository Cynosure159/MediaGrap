# Deployment and operations

[English quickstart](../README.md#try-docker) · [中文快速部署](../README-zh.md#docker-快速体验) · [Usage/configuration](usage.md) · [Distribution builds](distribution.md)

This guide describes the current checkout. The published `cynosure159/mediagrap:0.0.6` predates the AGPL/source-download/secure-cookie changes; do not infer that new settings work there. Pin a reviewed version or digest, not `latest`. No public GitHub source destination is assumed.

## Container and permissions

Production is one Go application process with an embedded Vue PWA in a non-root scratch image for linux/amd64 and linux/arm64. The image includes CA certificates and a minimal static ffprobe; Node, shells and build tools are not in the runtime. Probing launches a bounded subprocess when requested.

| Path | Access and lifetime |
| --- | --- |
| `/config` | Read/write; SQLite, settings, credentials, sessions, jobs, audit and integration state. Back up and protect. |
| `/cache` | Read/write; disposable cache and temporary data. |
| `/media/...` | Administrator-selected host mounts; read-only for discovery, read/write for NFO/artwork/rename. |

The default image UID/GID is **65532:65532**. On Linux, before launch:

```sh
sudo install -d -m 0750 -o 65532 -g 65532 runtime/config runtime/cache
```

These must be dedicated app directories, not your media root. Never solve a permission error by running the application as root, `chmod 777`, or recursive media ownership changes. Grant read/traverse access via host ACL/group policy; enable creation/rename/deletion only for intended writable sources. Numeric-user overrides require equivalent access to all state and media. There is no PUID/PGID entrypoint support. NAS ACLs, SELinux labels and root-squashed network mounts require deployment-specific verification; POSIX mode bits alone cannot prove writes will succeed.

The README's `docker run` uses `--mount` so a missing media host path is an error instead of creating an empty directory. In Settings add container-visible paths beneath `MEDIAGRAP_MEDIA_ROOTS` (comma-separated, default `/media`); mounting and allowlisting do not create sources. One logical source at the common mounted parent usually suffices. Source removal deletes configuration/index records, never mounted files.

For current-source builds, [export a reviewed context](distribution.md#build-contract) first. The repository [compose.yaml](../compose.yaml) requires `MEDIAGRAP_BUILD_CONTEXT`; it builds locally and is not a pull-only quickstart. Its `latest` image name is a local build label, not proof of publication. Review its all-interface host port mapping and supply explicit mounts/network isolation before running it. A bare `docker build .` is not a release build path.

## First setup and TLS

**The first visitor to a fresh instance can become administrator.** There is no bootstrap proof/network gate in the application. The default listener is `:8080`, not loopback. Isolate the host, Docker network and any proxy until setup completes. Publishing `127.0.0.1:8080:8080` does not prevent a peer container from reaching the container IP. On a remote NAS, keep setup private using a trusted SSH tunnel or an equivalent restricted network. Do not expose an unattended fresh instance publicly. Application hardening is a [high-priority deferred task](roadmap.md#highest-priority).

After setup, use HTTPS through a trusted reverse proxy for non-local access. Keep the backend accessible only to that proxy. For **current-source builds**, set `MEDIAGRAP_SECURE_SESSION_COOKIE=true` or `--secure-session-cookie=true`: setup, login and logout cookies become Secure even when the proxy-to-app hop is HTTP. Default false preserves local HTTP; requests presented to the handler as direct TLS always set Secure. The shipped listener itself serves HTTP; this is not a built-in TLS certificate configuration. `Forwarded` and `X-Forwarded-Proto` never enable Secure automatically. Do not enable Secure for a browser on plain HTTP or login will not persist. 0.0.6 must not be assumed to implement this setting.

Proxy the application at the origin root, including `/api`, job SSE, `/mcp` if used, and `/source`. Disable SSE buffering and allow long-lived connections. Preserve MCP Authorization, Origin and protocol headers. Do not hide the matching `/source` archive from remote users of a distributed/modified deployment. Browser PWA installation generally requires HTTPS or localhost; offline shell caching does not make metadata mutations offline-safe.

## Startup configuration

Flags override their corresponding environment defaults. These startup values are distinct from persisted operational Settings, whose saved values override provider environment fallbacks (see [usage](usage.md#provider-and-interface-settings)). Restart after changing startup environment.

| Environment | Flag | Default / meaning |
| --- | --- | --- |
| `MEDIAGRAP_CONFIG_DIR` | `--config-dir` | `/config` |
| `MEDIAGRAP_CACHE_DIR` | `--cache-dir` | `/cache` |
| `MEDIAGRAP_LISTEN` | `--listen` | `:8080`; host mapping does not change container listener |
| `MEDIAGRAP_LOG_FORMAT` | `--log-format` | `json`; `text` also accepted |
| `MEDIAGRAP_LOG_LEVEL` | `--log-level` | `info`; `debug`, `warn`, `error` |
| `MEDIAGRAP_SECURE_SESSION_COOKIE` | `--secure-session-cookie` | `false`; new current-source policy, not a 0.0.6 guarantee |
| `MEDIAGRAP_MEDIA_ROOTS` | — | `/media`; comma-separated allowed roots |
| `MEDIAGRAP_FFPROBE_PATH` | — | `ffprobe` natively, `/usr/bin/ffprobe` in image; `off` or `disabled` disables probing |

Complete provider/proxy variables are in [usage](usage.md#provider-and-interface-settings); Webhook/MCP deployment variables and key handling are in [integrations](integrations.md#configuration). Do not commit private environment files, keys or Compose overrides. Provider settings in SQLite are sensitive; they are not promised to be encrypted at rest.

The bundled ffprobe accepts local files only and enables AVI, FLAC, Matroska, MP3, MPEG-TS, MOV, Ogg and WAV demuxers. This does not expand the scanner's video extension allowlist. Unsupported/missing probes produce an unavailable state; file audit still works. Full source, build recipe and notices are included by [distribution builds](distribution.md).

## Backup and upgrade

There is **no built-in backup/restore command, automatic pre-migration backup or disk-space preflight guarantee**. Migrations run at startup. Operators must make a recoverable backup before upgrading:

1. Pause external automation and scheduled activity as appropriate. Wait for operations to settle, then stop MediaGrap; cancellation is not rollback.
2. Back up the complete config directory while stopped, including `mediagrap.db` and any remaining `-wal`/`-shm` files. Never copy only an active database file: committed data can still live in WAL. A separately arranged SQLite online-backup solution must itself be verified; MediaGrap supplies no such command.
3. Back up any separately mounted Webhook master key with the database, securely and with its permissions. A replacement key cannot decrypt old signing secrets.
4. Back up media-side NFO/artwork and any media affected by rename separately. `/cache` is disposable and is not a recovery backup.
5. Record the previous image digest and startup configuration. Obtain a reviewed new image, recreate the container with the same persistent mounts and verify readiness, source availability and permissions before scanning/writing.
6. Restore by stopping the app and replacing the complete config state with the matching backup, restoring the key and ownership, then starting the compatible image. Do not run two instances against that database. Downgrading an already migrated database is not promised; use the old image plus its matching backup.

A successful scan of an accessible empty mountpoint marks old items missing. Verify mounts after restart/upgrade first: the scanner cannot distinguish an accidentally unmounted empty directory from an intentionally empty source. Historical metadata remains in the database; source deletion is not a backup strategy.

Atomic replacement protects an individual supported file operation, not a series of files plus SQLite as one transaction. Old NFO contents/unknown XML fields are not backed up automatically. Review partial results and audit before retrying; arbitrary NAS failure cannot be globally rolled back.

## Health and troubleshooting

Use the published host port (8080 here):

```sh
curl --max-time 3 http://127.0.0.1:8080/healthz
curl --max-time 3 http://127.0.0.1:8080/readyz
docker logs --tail 100 mediagrap
```

Review/redact logs before sharing. `/healthz` checks HTTP liveness without acquiring SQLite. `/readyz` runs `SELECT 1` on the running process's shared pool with a two-second deadline, returning 200 or 503 and safe numeric pool statistics. Readiness statistics are unauthenticated so outages remain diagnosable; restrict probe access at a proxy if these aggregate counters are sensitive. Authenticated Operations also reports `database.pool`, version/build, actual WAL mode, migrations, cache/mount state and provider configuration/connectivity separately.

The image healthcheck runs `/mediagrap healthcheck`: a three-second HTTP readiness probe with proxies disabled, not a second database pool. Set `MEDIAGRAP_LISTEN` consistently; CLI-only listener overrides must also be reflected in healthcheck arguments/configuration. An unhealthy Docker status does not itself trigger the restart policy.

| Symptom | Check |
| --- | --- |
| Cannot create config/database | Dedicated state directories exist and are writable by UID/GID 65532; host ACL/SELinux/storage capacity. |
| Source rejected/empty | Existing host mount, container path, startup allowlist and read/traverse permissions. Do not scan a disconnected mount. |
| Login does not persist | Browser HTTPS versus explicit Secure-cookie setting; origin/cookie policy. Do not blindly trust forwarded headers. |
| Provider configured but fails | Authenticated fixed-target connection test, provider key, language, proxy/NO_PROXY and safe HTTP status. |
| `/source` returns 503 | Native/mismatched development archive; use a matching distribution build, not a live checkout fallback. |
| Liveness succeeds, readiness fails | Shared-pool/storage issue; collect repeated counters and safe timestamps before restart. |

`inUse=4`, `idle=0`, `saturated=true` means all four pool connections are held at that instant, not proof of deadlock. Increasing `waitCount` records acquisition waits; `waitDurationMs` counts completed waits, not the current age of blocked calls. A separate SQLite client succeeding cannot exclude main-pool starvation. Catalog code must consume/close rows before nested queries or filesystem work. Increasing workers/pool size is not a fix for nested connection acquisition.

A NAS syscall already underway may outlive context cancellation. Do not kill the process with SIGQUIT just to inspect it. Capture image digest, readiness samples, safe logs and storage status; deeper goroutine diagnostics need an explicitly instrumented build, not a publicly exposed profiler. No production NAS performance or universal atomicity claim is made.
