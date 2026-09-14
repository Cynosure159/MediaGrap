# Development and contributing

[Architecture/UI conventions](architecture.md) · [Security reporting](security.md) · [Distribution/CI](distribution.md) · [Roadmap](roadmap.md)

Contributions should be small, reviewable changes to the current implementation, not a revival of historical phase plans. Read both READMEs and the relevant maintained guide before changing behavior. Use the actual repository's contribution interface; no future GitHub namespace or private contact address is assumed here.

## Local setup

Prerequisites: **Go 1.26+**, **Node 24.15+**, **npm 11+**, Make; Python 3 and Docker for source-distribution checks. [go.mod](../go.mod), [web/package-lock.json](../web/package-lock.json) and [Dockerfile](../Dockerfile) define dependencies/build inputs. RTK is optional; standard commands below work without any developer-home configuration or plugin.

```sh
npm --prefix web ci
go mod download
MEDIAGRAP_LISTEN=127.0.0.1:8080 npm run dev
```

The root npm script runs `make dev`: Go API and Vite concurrently. Open the URL printed by Vite, which proxies `/api`, `/healthz`, `/readyz` and `/source` to `127.0.0.1:8080`. Running `npm --prefix web run dev` alone does not start Go. The explicit listener above binds the development API to loopback; the application's default remains all interfaces. Do not expose an uninitialized development instance to untrusted users.

Make reads ignored `.env.local` for your native media allowlist, for example:

```dotenv
MEDIAGRAP_MEDIA_ROOTS=/absolute/path/to/synthetic/movies,/absolute/path/to/synthetic/shows
```

Use existing absolute directories you own, add them as sources in Settings, and restart after allowlist changes. Defaults for native `make api` state/cache are `.local/config` and `.local/cache`, ignored by Git. These are sensitive state, not publishable fixtures. Tests must use temporary synthetic roots, never a developer's real library.

```sh
make build
# Builds web assets, copies them into Go embed input, creates bin/mediagrap.
MEDIAGRAP_CONFIG_DIR=./.local/config MEDIAGRAP_CACHE_DIR=./.local/cache \
  MEDIAGRAP_LISTEN=127.0.0.1:8080 ./bin/mediagrap
```

A fresh Go-only source build has a fallback page until the UI is built/copied. `make build` is a **development binary**, not a complete distribution: `/source` returns 503 without a valid baked source locator. Use the [exported-context Docker/artifacts path](distribution.md#native-builds-and-rebuild-limits) for distribution. `make docker-build` exports tracked files as a test-only snapshot; new untracked implementation files need individual reviewed `--test-extra` entries. It never implicitly includes local state.

## Checks

Run from the repository root:

```sh
# Fail when formatting differs, rather than merely printing a diff.
test -z "$(gofmt -l cmd internal)"
go vet ./...
go test -race ./...
python3 -m unittest discover -s scripts -p 'test_*.py'
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
git diff --check
```

There is **no separate frontend lint script**. Typecheck, Vitest and the production build are the actual gates; do not report a nonexistent lint command as passed. If a local Node version's experimental Web Storage conflicts with jsdom, use `NODE_OPTIONS=--no-experimental-webstorage npm --prefix web run test` and record why. `make format` changes Go files; use it deliberately, not during a read-only review.

Relevant safety checks and reproducible synthetic benchmarks:

```sh
go test ./internal/library -run 'TestCatalogSingleConnection|TestScan|TestRenamePlanIncrementalClassification' -count=1
go test ./internal/httpapi -run 'TestReadiness|TestLiveness|TestSource|TestSession' -count=1
go test ./internal/jobs -run TestConcurrentScanQueueAndRetry -count=1
go test ./internal/library -run '^$' -bench '^BenchmarkScanReaders(NFO)?$' -benchtime=1x -count=3
```

Use source-named tests for the changed boundary: NFO round trips/malformed input, filesystem conflict/symlink/EXDEV faults, migrations/WAL, provider mock responses and redaction, persisted-job cancellation/recovery, token/source scopes and source archive correspondence. Never run provider/network tests with private credentials just to fill a test gap. Local fixture timings are not NAS throughput, cold filesystem-cache measurements or idle application memory.

For UI changes, cover desktop and ~360–390px mobile flows, keyboard controls, translated empty/error/conflict states, PWA reload/update, and missing-target draft preservation. Preserve operation-token cleanup and selection-generation guards. Screenshots alone do not establish safe file behavior. Use only owned synthetic fixtures/localhost services; record cleanup and any unverified environment boundary.

The [CI workflow](../.github/workflows/ci.yml) runs Go/web and both Docker architectures on main/dev PRs/pushes and version tags. [Distribution guidance](distribution.md#github-checks-and-dockerhub-tags) lists stable required-check names, owner-configured rulesets, secrets and release constraints. Adding workflow YAML does not configure protection or authorize publication.

## Proposing a change

1. Describe the problem, expected behavior, data-safety impact and how to reproduce it with a minimal synthetic example. Discuss new behavior/architecture before a broad implementation.
2. Make one coherent change, preserve unrelated work, and use conventional commit titles (for example `fix(library): preserve saved metadata during scan`). Follow the hosting repository's branch/PR process; do not assume direct main pushes are allowed.
3. Add regression tests before or with the fix. Keep interfaces explicit and narrow, SQL transactions short, work bounded, and filesystem effects previewable/auditable. Do not add speculative frameworks or placeholder controls.
4. Update the canonical guide and both READMEs when entry-point behavior changes. Keep detailed documentation in `docs/`; short `.github` discovery pointers may link there. Do not reintroduce duplicate phase diaries or prototype dependencies.
5. Include changed scope, commands/results, migration/backup impact and residual risks in the review description. Separate checked behavior from future goals. Reviewers should confirm no secrets, local state, generated artifacts or unrelated files were staged.

Submit only material you have rights to contribute under the project's applicable license. Preserve original third-party copyright/license notices and identify deliberately imported dependencies/materials. A project AGPL declaration does not override dependency, provider-content or trademark terms. See [legal materials](legal/README.md). Never post credentials, real cookies, private media paths or sensitive database/log dumps; use the [security process](security.md#reporting-a-vulnerability) for vulnerabilities.

## Release boundary

Public source hosting, building a local image and distributing an image are distinct actions. Stable and RC release contexts use `--tag` with an existing owner-approved `vX.Y.Z` or `vX.Y.Z-rc.N` tag at HEAD and canonical tracked bytes; dirty snapshots are explicitly test-only. Dev/main pushes and PRs run checks only. Manual RC tag pushes must resolve to the current dev tip and publish the short RC version plus `preview`, never `latest`; stable tags must resolve to the current main tip. See [manual short-tag commands and source compatibility](distribution.md#manual-short-tag-release). Do not create tags, publish registries or change repository account settings as part of an ordinary patch without authorization. Regenerate source archives from the final reviewed tree, keep matching source available to remote users, and independently inspect dual-architecture build/source evidence before release. No offline whole-image reproducibility or legal certification is implied by passing CI.
