# Incremental scan batches (stage 2)

This records stage 2. [Stage 3 bounded readers and persisted tracking](scan-readers-and-tracking.md) retains these SQL batches while adding ordered read/parse concurrency; historical serial-hydration statements below describe stage 2 only.

## Behavior and boundaries

Scans still traverse the whole source, using the [directory snapshots](scan-directory-snapshots.md). They now prepare at most **200 video candidates** before committing one repository transaction. Filesystem stat/sidecar discovery and filename parsing finish before that transaction; eligible movie NFO hydration runs after commit, through the existing metadata service. There is no parallel scanner, whole-library identity cache, new SQLite writer, pool change, or durability pragma change.

`internal/library/scan_batch.go` owns the prepared candidate and SQL-only batch repository. Each batch queries only its candidate paths, updates changed media/sidecar/TV rows atomically, and bulk-updates `last_seen_at`/`missing=0` for unchanged videos. Directory snapshots remain directory-scoped; candidate/identity maps are batch-scoped. Memory still depends on the largest directory listing, pending child paths, and sidecars belonging to up to 200 candidates—not an accumulated library map. This is a count bound, not a fixed byte cap on unusually large directories or sidecar sets.

Migration `0019_scan_fingerprints.sql` adds empty-default fingerprints without rewriting existing metadata. Legacy rows are classified on their next scan. Incremental scans skip unchanged index rewrites and successful unchanged hydration. Full scans bypass the fingerprints and revalidate the index; they do **not** force overwrite saved metadata.

Successful application-managed video path updates (library rename plans and approved automation rename plans, including recovery) clear the scan fingerprint in the same SQL update. The next incremental scan reclassifies filename-derived TV titles, episode tokens and movie/TV membership even when video stat information is unchanged. Failed path updates and rolled-back automation transactions retain the previous fingerprint. Sidecar-only renames do not invalidate unrelated video classification.

TV show/season identities are resolved once per batch as needed, rather than upserted for every episode. Unchanged incremental episodes do not rewrite TV associations. Changed sidecars do not force unchanged TV upserts. Full scans repair derived title/year hints without changing saved TV titles. The existing root-layout grouping (`relative_path='.'`) remains unchanged: its last observed filename-derived show hint is reconciled once after successful traversal, including across batch boundaries; warm scans do not oscillate between early and late episode hints.

## Fingerprints and metadata priority

The versioned fingerprint hashes **stat information**, not file contents: video size, nanosecond mtime and mode; attributed sidecar paths/kinds/size/mtime/mode; and both movie NFO candidate states (same-basename `.nfo` and Kodi `movie.nfo` fallback, including multi-video folders). Additions, modifications and deletions therefore invalidate the relevant candidate even if the video itself is unchanged. Sidecar attribution and symlink/regular-file checks remain in place. Videos are revalidated with `Lstat` during preparation.

**Saved SQLite metadata still wins over local NFOs.** Detecting an external NFO change is not an automatic metadata synchronization operation. Imported records can later be manually edited while retaining `provider=nfo`; that string is not permission to overwrite them. Changed/new eligible movie NFOs are offered to the existing hydrator, which preserves any saved title (and its provider/locks). NFO deletion removes the sidecar from the index, not saved metadata. TV show/season artwork and local NFO read APIs retain their existing live-discovery and saved-metadata priority; this stage does not add TV metadata synchronization.

Missing SQLite movie metadata with an existing NFO is retried even when its fingerprint is unchanged. This covers interrupted post-commit hydration, invalid→repaired NFOs and metadata removed from SQLite. Movies without an NFO do not call the hydrator. A failed eligible NFO can be retried on each scan; invalid-NFO fallback remains non-blocking and does not rewrite files. The hydrator's existing type/size/XML checks and Kodi fallback are unchanged.

Limitations:

- Same-size replacements that preserve mtime and mode can evade incremental detection, especially on coarse-timestamp filesystems. Full scans bypass this stat comparison, but still preserve saved metadata. A regression test explicitly demonstrates this limitation.
- A scan is not an atomic filesystem snapshot. Names created after enumeration or files changed after stat may require another scan. There is no filesystem watcher or content hashing of videos/artwork.
- An accessible empty mountpoint remains indistinguishable from an intentionally empty source. NAS/NFS/SMB behavior and real media throughput were not measured locally.

## Cancellation, rollback and recovery

A failed batch rolls back its media, sidecar and TV writes together; earlier committed batches remain visible. Cancellation/read errors discard unflushed candidates. **Missing-file reconciliation and `last_scan_at` occur only after successful traversal, final batch/hydration handling, and a context check**, in a final SQL-only transaction. Historical sample/Extras entries are retired only there, retaining historical-parent exclusion. Movies, episodes and empty shows disappear from catalogs after a successful scan, not on failure.

The durable job worker's interrupted/cancelled/retry policies are unchanged. Retrying starts a fresh traversal/seen epoch, reuses committed fingerprints and restores missing hydration instead of requiring an in-memory cache. Tests cover a 401-video second-batch rollback, sidecar/TV-trigger failures, cancellation/read errors after a committed batch, and an interrupted persisted job retried through a fresh worker.

## Reproducible local measurements

```sh
rtk proxy go test ./internal/library -run '^$' -bench '^BenchmarkScanIncremental$' -benchtime=1x -count=1 -timeout=10m
rtk proxy go test ./internal/library -run 'TestScanBatch|TestScansReconcile|TestScanSkippedHistory|TestScanDirectorySnapshotsSidecarAttribution' -count=1 -v
```

Measured on local macOS/darwin arm64, Apple M4 Pro, 2026-09-12. Stage 1 was copied before stage-2 edits into an isolated temporary source tree; **the same benchmark and SQLite driver observer** were run against both versions. Both retain the application's WAL/pool/durability settings. Fixture creation/migrations/seeding are outside the Go benchmark timer. Each subcase uses a fresh database and 1,000 or 10,000 seven-byte videos with seven-byte NFOs, 80% movie names/20% episode names; nested layout has 100 videos per child directory. There is no metadata hydrator, ffprobe, provider or UI in these index-only timings.

“Cold” means **empty index**, not cold OS cache. Warm/change cases first seed the index outside the timer; 1%-change overwrites the first 1% of videos with a different-size fixture. All figures below are single iterations, serially run with uncontrolled cache state—not statistically significant estimates or production NAS performance. The test-only SQL observer adds overhead to both versions. `SQL` counts actual submitted statements; `tx` counts actual **explicit `BeginTx` calls**, including final reconciliation. Stage-1 implicit autocommit writes are **not** included in `tx`; it is not a count of disk flushes or all SQLite internal transactions.

| Layout / phase (10,000 videos) | Stage 1 elapsed | Stage 2 elapsed | SQL: stage 1 → 2 | Explicit tx: stage 1 → 2 | Allocated bytes: stage 1 → 2 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Flat / cold | 1.558 s | 0.676 s | 44,003 → 40,077 | 10,001 → 51 | 117,680,576 → 130,193,512 |
| Flat / warm | 1.648 s | 0.152 s | 44,003 → 104 | 10,001 → 51 | 117,619,104 → 53,809,640 |
| Flat / 1% change | 1.519 s | 0.186 s | 44,003 → 404 | 10,001 → 51 | 118,346,192 → 56,504,600 |
| Nested / cold | 1.728 s | 0.700 s | 44,003 → 40,453 | 10,001 → 51 | 119,543,208 → 133,352,960 |
| Nested / warm | 1.789 s | 0.162 s | 44,003 → 103 | 10,001 → 51 | 120,248,688 → 55,663,528 |
| Nested / 1% change | 1.692 s | 0.170 s | 44,003 → 403 | 10,001 → 51 | 121,157,128 → 58,172,472 |

Cold cumulative allocations **increase** with fingerprint/preparation work; warm allocations and SQL fall substantially. Allocated bytes are cumulative Go allocations, not retained heap or peak RSS. The complete 1,000-video results are retained in the raw evidence; deterministic all-movie tests separately observe 3,008/13/33 SQL statements and 6 explicit transactions for cold/warm/1%-change scans, with zero warm sidecar/TV rewrites and all 1,000 videos seen in the new epoch.

Candidate-size exploration used isolated copies with only the batch constant changed, the same final code and 10,000-flat fixture:

| Candidates | Cold | Warm | 1% change | Explicit tx |
| --- | ---: | ---: | ---: | ---: |
| 100 | 0.659 s | 0.158 s | 0.167 s | 101 |
| **200** | 0.676 s | 0.152 s | 0.186 s | 51 |
| 300 | 0.701 s | 0.183 s | 0.177 s | 35 |

200 is a bounded middle choice with fewer commits than 100, not a statistically proven optimum. The 1%-change sample at 100 was faster than 200; no monotonic runtime claim is made.

Separate `/usr/bin/time -l` runs of compiled test binaries measured whole-process maximum RSS of **38,633,472 bytes (stage 1)** and **38,567,936 bytes (stage 2)** for the 10,000-flat warm subcase—effectively similar in these single runs. This includes **fixture setup, initial indexing, test runtime and cleanup**, not scan-only peak memory. See `rss-stage1.txt` and `rss-stage2.txt` in the acceptance evidence; these values must not be equated with idle application memory or production RSS.

## Actual browser validation

Ego Lite TaskSpace16/p1 was verified agent-owned and used only with a temporary localhost server containing the production frontend build. Desktop 1440×1000 and mobile 390×844 exercised six real UI scan jobs: cold/warm scans, NFO/artwork-only changes, movie deletion, episode deletion, empty-show removal and restoration. Observed totals were 220→219 movies and 2→1 shows; the remaining show retained E02 but not E01. The missing selected movie inspector cleared its old plot. New/repaired NFOs hydrated, a changed previously imported NFO did not overwrite saved metadata, and removed artwork disappeared. All 227 original temporary files were restored at their original paths with identical SHA-256 hashes. The owned server stopped and the TaskSpace was finished.

Screenshots/snapshots, fixture manifests, raw benchmark outputs, final validation logs and patches are preserved in the stage-2 acceptance artifact's `stage-2-evidence/` directory. No user media, production data/server, commits, pushes or publishing were involved. The final TV hint compatibility correction affects root-layout or stale indexed hints only; supervisor approved reuse of these conventional-directory browser checks, with new root-layout/cross-batch/full-repair Go regressions. No frontend implementation changed. Frontend lint remains unconfigured; typecheck, tests and production build are required and recorded separately.

The rename review correction was additionally checked in a new owned Ego Lite TaskSpace17/p1 using four isolated temporary videos and a fresh localhost application. Real UI previews/applies and three incremental scans changed S01E01 Old to S02E03 New (moving it into the root-layout show), then converted a movie to S02E04 Converted. Desktop/mobile catalogs showed 2→1 movies, 1→2 shows, and the new episode titles/tokens without old selected details after navigation. Screenshots and logs are in `stage-2-rename-evidence/` alongside the correction acceptance artifact. All four fixture paths and SHA-256 hashes were restored, and the owned server/space stopped. At 390×844 the existing bottom navigation overlapped the rename execute button and the long source path placed Scan outside the viewport; documented keyboard focus/Enter activated those UI controls. This verifies the scan/classification behavior, not unobstructed mobile pointer usability; no layout fix is included in this backend-only correction.
