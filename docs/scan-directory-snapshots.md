# Directory-scoped scan snapshots (stage 1)

This document records the accepted stage-1 boundary and measurements. The subsequent [stage-2 incremental batches](incremental-scan-batches.md) build on these snapshots; the historical no-batching statements and original timings below describe stage 1 only.

## Boundary

`internal/library/scan_snapshot.go` owns directory enumeration, a sorted sidecar-name snapshot, and file-first traversal. `service.go` retains video/TV indexing, metadata hydration, and final missing-file reconciliation. `supplemental.go` applies the same regular-main-video test to already enumerated parent entries; historical direct-parent identity still excludes orphaned Extras.

The scan reads each visited directory once. A snapshot holds only that directory's entries, non-sample video count, and supported sidecar names. Sorted literal-prefix lookup avoids scanning every sidecar for every video in a flat directory. Single-video directories still attribute all supported sidecars. `Lstat` still rejects symlink/non-regular sidecars at attribution time; no NFO parsing, validation, provenance, or hydration-priority rules changed.

Direct files are processed before descending, and the snapshot is discarded before recursion. Pending child paths on the ancestor stack remain, so memory depends on directory width/depth, not a retained index of the whole library. An exceptionally large single directory still requires a correspondingly large listing. Directory contents are not an atomic filesystem view; newly added names after enumeration are discovered on the next scan. Read APIs build fresh snapshots; scan snapshots are never shared between requests/scans.

No batching, concurrency, database pragma changes, or actual incremental change detection is included. Filesystem discovery happens before each existing sidecar write transaction. The one SQLite writer and durability settings are unchanged.

A narrowly related safety correction removes early `missing=1` writes for samples/Extras: skipped historical rows are retired only by the existing successful final reconciliation. Cancellation or traversal error does not advance `last_scan_at` or hide those rows. Metadata and audit history remain stored.

## Reproducible local benchmark

```sh
rtk proxy go test ./internal/library -run '^$' -bench '^BenchmarkScanDirectorySnapshots$' -benchtime=1x -count=1 -timeout=15m
rtk proxy go test ./internal/library -run 'TestScanDirectorySnapshots|TestDirectorySnapshot|TestScanSkippedHistory' -v
```

Measured on local macOS/darwin arm64, Apple M4 Pro, 2026-09-12. Each subcase starts a fresh temporary library and migrated SQLite database. Setup is outside the timed region. Videos and NFOs contain seven-byte fixture payloads; 80% of names are movies, 20% TV episodes. Flat cases put all videos in one directory; nested cases use 100 videos per child directory. No metadata hydrator/provider or ffprobe runs in this index-only benchmark. WAL/durability defaults are unchanged.

Baseline used `cc107926d19bff6e1b86355fcabaaccc8b2f5cf4` production code with the same benchmark added before implementation. Optimized results below are the serial rerun after other checks/local browser services stopped. Both are single timed iterations, not statistical estimates; filesystem cache state was uncontrolled. A preliminary optimized run overlapped unit tests and is not used in this table.

| Layout / videos | Baseline elapsed | Optimized elapsed | Baseline allocated bytes | Optimized allocated bytes |
| --- | ---: | ---: | ---: | ---: |
| Flat / 100 | 41.226 ms | 25.524 ms | 3,834,736 | 703,600 |
| Flat / 1,000 | 1.030 s | 0.161 s | 305,574,320 | 6,395,456 |
| Flat / 10,000 | 96.472 s | 1.437 s | 35,269,755,656 | 63,290,336 |
| Nested / 100 | 35.012 ms | 24.252 ms | 3,749,240 | 746,480 |
| Nested / 1,000 | 0.261 s | 0.166 s | 37,212,816 | 6,399,232 |
| Nested / 10,000 | 2.817 s | 1.486 s | 371,652,680 | 64,402,464 |

Allocated bytes are **cumulative Go allocations per scan**, not peak RSS or retained heap. These tiny local fixtures do **not** measure production NAS/NFS/SMB latency, real media I/O, valid NFO hydration cost, network/provider throughput, cold caches, or full application memory. Those environments remain unmeasured.

Deterministic enumeration tests assert flat counts of **1/1/1** and nested counts of **2/11/101** for 100/1,000/10,000 videos. A service integration test verifies one enumeration per visited directory while recording literal/overlapping-prefix sidecars, sample-aware directory assets, symlink exclusion, and TV ownership. The baseline code performs one traversal enumeration per directory plus one sidecar enumeration per video (a code-derived count, not a baseline syscall trace). Historical parent/sample retirement, error/cancellation, and sidecar replacement/rescan freshness have explicit regression tests.

The service integration budget shares an instance-scoped enumeration dependency with live sidecar discovery, not just the directory walker. `TestScanDirectorySnapshotsSidecarAttribution` asserts exactly four directory reads for five indexed videos on both full scan and rescan, then separately verifies that five live sidecar lookups cause five reads through the same counter. This guards against accidentally returning to per-video discovery. An isolated temporary source-copy mutation replacing snapshot attribution in `scanVideo` with `recordSidecars` failed this test with nine reads (root: 3, Solo: 2, show: 1, season: 3) instead of four; restoring the optimized code passed. No exported API or metadata-hydrator boundary changed.

## Isolated browser validation

Ego Lite TaskSpace15/p1 was verified agent-owned and blank, then used only with a temporary localhost Go application and Vue development server. Desktop 1440×1000 and mobile 390×844 both exercised real catalog scan controls and movie/TV details. The production frontend build also passed separately.

- Initial/restored catalog: 30 movies, two shows with two and one episodes.
- Desktop rescan after temporary video moves: 29 movies; selected deleted movie shows not-found rather than stale metadata. Deleting a show's final episode reduces the show count to one and clears its inspector.
- Mobile rescan: restored 30 movies, then 29 movies after temporary deletion; the two-episode show becomes one episode, and reopened details omit the removed episode.
- Valid local NFO title/year/plot hydration and invalid-NFO fallback were observed; switching selections did not retain the previous plot.
- The job center showed five successful real library scan jobs. All 37 fixture files, including NFO, sample, and Extras, were restored to original paths with matching SHA-256 hashes. No user media was scanned or modified. Owned localhost services were stopped and the browser task was finished.

Raw benchmark outputs, enumeration/test logs, browser snapshots/screenshots, and fixture manifests accompany the stage-1 acceptance artifact. These checks cover the bounded stage only, not later scan optimizations.
