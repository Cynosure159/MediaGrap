# Phase 4D media inspection and file audit

Phase 4D adds a read-only movie inspection slice. It does not rename, move, delete, overwrite, or open files on the host.

## API behavior

- `GET /api/v1/media/{id}/inspection` returns optional ffprobe results plus a fresh filesystem audit.
- `POST /api/v1/media/{id}/naming-preview` accepts `{ "pattern": "${title} (${year})" }` and returns an in-memory preview only. It is a read operation despite using POST for the structured template input; there is deliberately no apply endpoint in Phase 4D.
- Both endpoints require an authenticated session. The preview endpoint performs no mutation and therefore does not require a CSRF token.

## Probe and cache

The server runs ffprobe directly, without a shell, using `-v quiet -print_format json -show_format -show_streams`, a 30-second context timeout, and an 8 MiB output limit. It maps video, audio, subtitle, HDR, resolution, language, channel layout, duration, container and bitrate fields into provider-neutral response types.

Successful probe output is stored in `media_probe_cache`. The cache key is the media item ID plus the current file size and nanosecond modification time. Concurrent requests for the same item are serialized, and the server checks the fingerprint again after ffprobe exits so a file changed during inspection is never cached under the old fingerprint. Failed or unavailable probes are not cached.

## Filesystem safety

Before probing, MediaGrap resolves the source root and media parent directory and confirms the resolved parent remains inside the root. The media leaf and each Sidecar are inspected with `os.Lstat`; symlink leaves are reported but never opened or followed. The audit reports only source-relative paths.

Associated assets include same-basename NFO, image and subtitle files, plus conventional Kodi directory-level assets when the directory contains exactly one video. Regular NFO files receive a bounded XML syntax check. Read-only permission bits are warnings rather than invalid-file errors.

## Naming preview boundary

The initial template supports `${title}`, `${originalTitle}`, `${year}`, `${resolution}`, `${videoCodec}`, and `${audioCodec}`. Templates that contain `/` or `\\`, unknown variables, or an empty result are rejected. Unsafe filename characters are replaced, destination existence is reported as a conflict, and same-basename language/edition suffixes are preserved.

The preview never persists a plan and never calls rename or move. Directory templates, cross-filesystem moves, case-folding rules, retries, cancellation, revalidation at apply time, and audit records remain Phase 5B work.

## UI placement

The workspace keeps the five standard tabs from the Stitch design. Overview displays compact real probe pills and a real stream summary. File Audit contains the detailed file table, video/audio/subtitle stream groups, and the naming Dry-Run comparison. Missing ffprobe is shown as unavailable rather than replaced with filename guesses or static defaults.
