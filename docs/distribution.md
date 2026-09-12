# Source and image distribution

[Deployment](deployment.md) · [Development](development.md) · [Legal materials](legal/README.md)

This describes the current build changes, **not the already-published DockerHub 0.0.6 image**. No GitHub repository address has been chosen here. The module remains `github.com/mediagrap/mediagrap`; the actual DockerHub destination defaults to `cynosure159/mediagrap`.

## Build contract

Release sources must come from a clean tracked tree whose existing `vSemVer` tag identifies HEAD. No local environment, credentials, config/data/cache/runtime roots, Git metadata, node_modules or generated archive recursion is included. The exporter fails on unsafe paths/symlinks or dirty releases. It neither creates tags nor commits/pushes anything.

```sh
# TAG must already name an owner-approved release; this does not create it.
python3 scripts/release-context.py --tag "$TAG" --output /absolute/new/context
# Local build only; no registry upload.
docker build -t mediagrap:review /absolute/new/context
python3 scripts/verify-image.py mediagrap:review --output /absolute/new/evidence
```

Use `--test-snapshot` instead of `--tag` for explicitly non-release tracked working-tree tests. New implementation files can be listed individually with `--test-extra path`; this is never allowed for a release. Test archives carry `testOnly: true` and a `test-<commit>` version. Untracked files are never automatically swept into the archive. `make docker-build` builds a tracked test snapshot. Export output must not already exist and must be outside the checkout.

The Docker build refuses missing input manifests, mismatched source hashes/version/commit, missing runtime source/notices or changed pinned dependencies. It embeds the archive into the Go binary before linking. `/source` serves only this immutable same-build download, without authentication, and rejects queries, encoded paths and suffixes. There is no runtime filesystem/proxy/source-tree fallback. The PWA does not precache the archive or intercept `/source` navigation. This endpoint must remain reachable by remote users of distributed/modified deployments.

`MANIFEST.json` records source file hashes/sizes; the embedded download manifest records archive SHA256 and bytes. Image verification measures image size and checks archive identity against `/api/v1/system/info`, archive hashes, legal materials, HEAD delivery and non-root startup using isolated synthetic state. `/usr/share/mediagrap` also contains the project license and ffprobe/runtime source material. [Legal inventory and exact hashes](legal/README.md) explain component licenses without replacing upstream notices.

## Native builds and rebuild limits

`make build` produces a development binary. Missing or mismatched baked source returns HTTP 503; it must not be presented as a complete distribution. For a Linux standalone binary with the same embedded archive, export Docker's `artifacts` target (`docker build --target artifacts --output type=local,dest=/absolute/new/artifacts /absolute/context`). Retain the accompanying ffprobe and legal/source materials. A Darwin/Windows release pipeline is not provided.

The archive contains the project build scripts, npm lock, Go module zip sources, runtime npm distributions (build inputs), pinned Vue/router preferred upstream source archives, nested bundle notices, FFmpeg archive and build/configuration logs, runtime component sources/notices and actual APK/compiler versions. Rebuilding the frontend requires npm-registry access and the Node toolchain; rebuilding Go requires the recorded toolchain/module availability or restoring supplied module sources. Builder base-image/package availability remains required. No fully offline or bit-for-bit whole-image reproducibility guarantee is claimed. The supplied dependency material is inventoried in the [legal guide](legal/README.md), not a claim to vendor every development tool, and can be substantial; measure archive/image bytes for each release rather than promising a sub-100-MiB image. Do not substitute an upstream URL for supplying the FFmpeg material already inside the archive. Recreate a verified build context with `python3 scripts/rebuild-source.py source.tar.gz --output /absolute/new/rebuild`, then build that context with its archived Dockerfile. Review downloaded source before executing its build scripts. The npm packages are private build components; their package.json versions are not the application release version, which comes exclusively from the verified tag/input manifest.

Rebuild the standalone ffprobe from its supplied source using `scripts/build-ffprobe.sh` in the recorded Alpine toolchain environment; replace `/usr/bin/ffprobe` in a derived image to use a modified build.

## Vue and router preferred source

`third-party/npm` preserves the exact npm distributions used as build inputs, including unmodified bundles/declarations; these are not the preferred TypeScript source trees. `third-party/frontend-upstream` additionally supplies the complete, unchanged Vue 3.5.42 and vue-router 4.6.4 upstream archives, with origin URLs, SHA256, source-directory-to-npm-package mappings and the verified devtools-api 6.6.4 notice in `inventory.json`. The same pins are reviewed in [frontend-sources.json](legal/frontend-sources.json). Downloads fail closed on a checksum or package-version mismatch. Tag URLs are checksum-pinned, not signature attestations.

To rebuild those dependencies, first verify their SHA256 against the inventory and unpack in a disposable directory. Use the Node and pnpm versions declared by each archived root `package.json` (Vue: Node ≥20, pnpm 11.19.0; router: Node ≥22.18, pnpm 10.18.2). With those toolchains available, from the respective extracted root:

```sh
# Vue core-3.5.42: scripts/build.js and rollup.config.js
pnpm install --frozen-lockfile
pnpm run build
# Router router-4.6.4: packages/router/tsdown.config.ts
pnpm install --frozen-lockfile
pnpm --filter vue-router run build
```

Vue outputs are under `packages/*/dist`; router outputs are under `packages/router/dist`. Review upstream lifecycle scripts before installing. These recipes use the supplied upstream lock/build files and require registry/toolchain access; MediaGrap's normal build continues to consume the exact locked npm releases. To experiment with modified dependencies, replace the relevant npm package build inputs in an owned development checkout and rebuild the frontend; do not label that modified output as the unmodified pinned release. This is not a bit-for-bit upstream npm publication guarantee.

## GitHub checks and DockerHub tags

The workflow runs on PRs targeting **main and dev**, pushes to both branches and `v*` tags. Stable check names for owner-configured branch protection/rulesets:

- `Go and web checks`: fail-on-output gofmt, go vet, Go race tests, release-script tests, actual web typecheck/test/production build.
- `Docker (amd64)` and `Docker (arm64)`: image build and same-build source/non-root runtime verification.

Configure these required checks separately on both branches; a repository file cannot enable protection. Untrusted PRs use `ubuntu-latest`, read-only repository permissions, and no registry secrets or `pull_request_target`. Actions use upstream version tags rather than unverified invented SHA pins; review and pin verified upstream SHAs separately if desired. There is no separate frontend lint script; typecheck/tests/build are the current frontend gates.

Tag publication depends on every check. It validates vSemVer, tag/commit/version correspondence and non-test candidates **before login**, then publishes the exact saved/tested amd64/arm64 images, not an independently rebuilt candidate. The default image is `cynosure159/mediagrap`, configurable via repository variable `DOCKERHUB_IMAGE` (`namespace/image`). Owner-configured secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` with only necessary push access. Never commit these values. Version tags omit the initial `v`; prereleases never update `latest`. Publication is serialized without canceling an active publish. GitHub concurrency can replace older pending runs; re-run an intentionally skipped tag workflow if needed.

Each architecture uploads its source archive, inventory/checksums, ffprobe diagnostics, runtime verification and (for tags) tested image as a GitHub Actions artifact retained 30 days. No GitHub Release is automatically created or attached. These temporary CI artifacts are not the long-term source offer: the same-build `/source` archive remains embedded in every built image/binary. Retain source artifacts alongside any separately redistributed binaries. Registry tags can be moved by privileged owners; consumers should pin image digests. No account settings, tags, publication or deployment were executed by adding this workflow.

## Release acceptance

Before enabling tag publication, independently review both architecture evidence and perform a clean rebuild from the downloaded source archive, including a modified/relinked ffprobe check. This implementation stage is not a legal certification, historical relicensing, asset provenance audit or provider-terms review. The first-admin bootstrap behavior remains unchanged: isolate setup before untrusted access; see [deployment guidance](deployment.md#first-setup-and-tls). Secure cookies alone do not prevent first-visitor takeover.
