# Database starvation diagnostics (0.0.2)

## What this release fixes

Movie catalog queries previously fetched source roots while their result set still held a database connection. TV detail did the same before consuming episode rows. With all four connections held by such requests, every request could wait for another connection indefinitely. Catalog/source rows are now consumed and closed before nested database queries or media filesystem access. `TestCatalogSingleConnection` reproduces both failures on the previous implementation with a one-connection pool and passes with the repair. This is a demonstrated code defect; without a production goroutine trace it does not prove that every server hang had this cause.

Sidecar discovery now reads a directory once instead of up to three times per media item, with literal filename matching (including brackets). Sidecar index replacement uses one short transaction, after filesystem discovery, and propagates persistence failures. There is no library-wide in-memory cache. Scan queue insertion atomically excludes another queued/running scan for that source; browser scans return HTTP 409 on duplicates. Retrying a scan also checks for another active scan. This covers jobs of kind `scan`, not automation wrapper jobs. A cancelled running handler may still be finishing cooperative cleanup. One worker remains deliberate; increasing worker counts or connection limits is not a fix for nested acquisition.

## Probes

Use the published HOST port, which may differ from container port 8080:

```sh
curl --max-time 3 http://127.0.0.1:HOST_PORT/healthz
curl --max-time 3 http://127.0.0.1:HOST_PORT/readyz
```

`/healthz` only confirms HTTP liveness. `/readyz` executes `SELECT 1` against the **running process's shared pool** with a two-second context deadline. It returns HTTP 200 or 503 plus safe numeric pool statistics (no paths, SQL text, credentials, or metadata). These statistics are deliberately available without database-backed authentication so they can be collected during an outage. Restrict probe routes at your reverse proxy if exposing aggregate operational counters is undesirable. The authenticated operations status also includes `database.pool`.

The container `healthcheck` command now probes the running HTTP readiness endpoint with a three-second client timeout and proxies disabled. Previously it opened a new process-local SQLite pool and could incorrectly report healthy while the main pool was exhausted. Configure the listener through `MEDIAGRAP_LISTEN` so the application and healthcheck agree; CLI-only listener overrides must also be supplied to healthcheck configuration. Docker marks failures unhealthy; Docker restart policies do not automatically restart unhealthy containers.

Interpret repeated samples:

- `maxOpenConnections=4`, `inUse=4`, `idle=0`, `saturated=true`: all connections are checked out at the sampling instant.
- Increasing `waitCount`: acquisition attempts have had to wait. Increasing `waitDurationMs` measures completed waits, not the continuously growing age of currently blocked waits.
- Saturation alone does not prove deadlock. It can be transient load, a SQLite write lock, slow filesystem work while a connection is held, or nested acquisition.
- Liveness succeeds but readiness fails: HTTP is alive but the main pool/database cannot complete its probe within the budget.
- An independent SQLite client succeeding cannot rule out main-process pool starvation.

A failed readiness request logs `database readiness check failed` with pool counters. The read-only system summary has a two-second cooperative context budget. There is intentionally no blanket timeout on file-writing/scraping handlers: a cancelled context is not a promise that an already-running NAS syscall or atomic file publication was interrupted. No unbounded background goroutines or automatic restart/data repair are used as timeout workarounds.

If a hang remains, capture readiness samples and timestamps, container image digest, safe logs, and storage status before restarting. Distinguishing a connection holder stuck on SQLite from a NAS syscall requires a safely collected goroutine profile in an explicitly instrumented build; this release does not publicly expose pprof or accept diagnostic signals that terminate the process. Do not use SIGQUIT on production merely to inspect it.

## Repeatable validation

```sh
go test ./internal/library -run TestCatalogSingleConnection -count=1
go test ./internal/httpapi -run 'TestReadinessDetectsPoolExhaustionAndRecovery|TestLivenessDoesNotAcquireDatabase' -count=1
go test ./internal/jobs -run TestConcurrentScanQueueAndRetry -count=1
go test ./internal/app -run TestCheckReadinessUsesRunningServer -count=1
go test -race ./...
```

The exhaustion test reserves the only main-pool connection, verifies deadline/503 and saturated counters, opens the same database independently, then releases the held connection and verifies readiness recovery. All fixtures use temporary directories, not actual media. No production load-test or media mutation is required.
