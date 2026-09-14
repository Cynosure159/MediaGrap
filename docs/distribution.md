# Source and image distribution

[Deployment](deployment.md) · [Development](development.md) · [Legal materials](legal/README.md)

This describes the current build changes, **not the already-published DockerHub 0.0.6 image**. Matching source is pinned to `Cynosure159/MediaGrap` GitHub Release assets. The module remains `github.com/mediagrap/mediagrap`; the actual DockerHub destination defaults to `cynosure159/mediagrap`.

## Build contract

Release sources must come from a clean tracked tree whose existing `vX.Y.Z` (stable) or `vX.Y.Z-rc.N` (RC) tag identifies HEAD. `N` is a positive decimal integer without leading zeros; version components are nonnegative integers without leading zeros. Use `--tag` for both channels; the former `--preview` export mode is removed. The exporter checks local tag/HEAD identity and canonical tracked bytes; CI separately checks the live tag and required remote branch tip before building and publishing. No local environment, credentials, config/data/cache/runtime roots, Git metadata, node_modules or generated archive recursion is included. The exporter fails on unsafe paths/symlinks or dirty releases. It neither creates tags nor commits/pushes anything.

```sh
# TAG must already name an owner-approved release; this does not create it.
python3 scripts/release-context.py --tag "$TAG" --output /absolute/new/context
# Local build only; no registry upload.
docker build -t mediagrap:review /absolute/new/context
docker build --target artifacts --output type=local,dest=/absolute/new/artifacts /absolute/new/context
# Owned HTTP fixture; does NOT prove that a public Release asset exists.
python3 scripts/verify-image.py mediagrap:review \
  --source-directory /absolute/new/artifacts/distribution --output /absolute/new/evidence
```

Use `--test-snapshot` instead of `--tag` for explicitly non-release tracked working-tree tests. New implementation files can be listed individually with `--test-extra path`; this is never allowed for a release. Test archives carry `testOnly: true` and a `test-<commit>` version. Untracked files are never automatically swept into the archive. `make docker-build` builds a tracked test snapshot. Export output must not already exist and must be outside the checkout.

The Docker build refuses missing input manifests, mismatched source hashes/version/commit, missing runtime source/notices or changed pinned dependencies. It generates the archive before linking but keeps it outside the final image. Only a small HTTPS locator, SHA256 and architecture are linked into the binary alongside version/commit. `/source` GET/HEAD returns 307 to the exact GitHub Release asset, exposes `X-Source-SHA256`, and rejects queries, encoded paths and suffixes. Missing/invalid locators return 503; the server does not check external availability on each request. There is no runtime filesystem/proxy/source-tree fallback. The PWA does not precache the archive or intercept `/source` navigation. This endpoint must remain reachable by remote users of distributed/modified deployments.

The pinned Alpine CA source is downloaded from the official `distfiles.alpinelinux.org/distfiles/v3.22/` mirror rather than the GitLab archive endpoint, which can reject unattended builds with HTTP 418. The archive bytes, SHA256 check and installed package pins are unchanged; the complete source and notices remain in the external matching-source bundle. Network access to this mirror is required even when rebuilding from downloaded project source.

`MANIFEST.json` records version/commit/architecture and all source file hashes/sizes; the exported `distribution/manifest.json` records archive SHA256, bytes, asset name and URL. Image verification measures image size and checks archive identity against `/api/v1/system/info`, archive hashes, legal materials, GET/HEAD redirects and fixture downloads and non-root startup using isolated synthetic state. `/usr/share/mediagrap` retains the project license, runtime notices, ffprobe diagnostics/configuration and component hashes. The duplicate FFmpeg, musl and CA source archives live in the external bundle, not the runtime image. Verification checks the exact image ID through an owned unstarted container export, matches all notices/component bytes against that bundle and rejects source archive files in the runtime. Started-container non-root HTTP checks are separate; any started CA checksum difference is recorded with container/image IDs and Docker version, not accepted as a change to distributed image bytes. [Legal inventory and exact hashes](legal/README.md) explain component licenses without replacing upstream notices.

## Native builds and rebuild limits

`make build` produces a development binary. Missing or invalid baked source locators return HTTP 503; it must not be presented as a complete distribution. For a Linux standalone binary with the same pinned external locator and separately exported archive, export Docker's `artifacts` target (`docker build --target artifacts --output type=local,dest=/absolute/new/artifacts /absolute/context`). Retain the accompanying ffprobe and legal/source materials. A Darwin/Windows release pipeline is not provided.

The archive contains the project build scripts, npm lock, Go module zip sources, runtime npm distributions (build inputs), pinned Vue/router preferred upstream source archives, nested bundle notices, FFmpeg archive and build/configuration logs, runtime component sources/notices and actual APK/compiler versions. Rebuilding the frontend requires npm-registry access and the Node toolchain; rebuilding Go requires the recorded toolchain/module availability or restoring supplied module sources. Builder base-image/package availability remains required. No fully offline or bit-for-bit whole-image reproducibility guarantee is claimed. The supplied dependency material is inventoried in the [legal guide](legal/README.md), not a claim to vendor every development tool, and can be substantial; measure archive/image bytes for each release rather than promising a sub-100-MiB image. Do not substitute an upstream URL for supplying the FFmpeg material inside the matching public archive. Recreate a verified build context with `python3 scripts/rebuild-source.py source.tar.gz --output /absolute/new/rebuild`, then build that context with its archived Dockerfile. Review downloaded source before executing its build scripts. The npm packages are private build components; their package.json versions are not the application release version, which comes exclusively from the verified tag/input manifest.

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

The workflow runs on PRs targeting **main and dev**, pushes to both branches and short stable/RC version tag pushes. GitHub tag globs select numeric `vX.Y.Z` / `vX.Y.Z-rc.N` shapes; script validation rejects noncanonical spellings. Stable check names for owner-configured branch protection/rulesets:

- `Go and web checks`: fail-on-output gofmt, go vet, Go race tests, release-script tests, actual web typecheck/test/production build.
- `Docker (amd64)` and `Docker (arm64)`: image build and same-build source/non-root runtime verification.

Keep all three required checks and one approving review on both branches, with no bypass. These are owner-managed protections; this change does not modify permissions or rulesets, and a repository file cannot enable protection. Verification/PR jobs have read-only repository permissions, no registry credentials and no `pull_request_target`. Docker jobs use native `ubuntu-24.04` (amd64) and `ubuntu-24.04-arm` (arm64), with no QEMU setup; hosted ARM allocation remains an environment gate. Actions use upstream version tags. There is no separate frontend lint script.

Publication depends on all three checks and uses only the saved, tested images. Only the gated publication job has repository write access; `github.token` is scoped to source publication and DockerHub credentials to registry login. The default image is `cynosure159/mediagrap`, configurable via `DOCKERHUB_IMAGE` (`namespace/image`). Owner-configured secrets are `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`; never commit their values.

- Dev/main pushes and all PRs run checks only, using test-only source snapshots. They do not save publication images, create Releases or log in/push to DockerHub. Superseded dev checks are canceled.
- Only an explicitly pushed `vX.Y.Z-rc.N` tag at the exact triggering commit **and current remote dev tip** publishes an RC. It creates a GitHub prerelease at that existing tag, immutable image `X.Y.Z-rc.N` (plus `-amd64` / `-arm64` implementation tags), and mutable `preview`. RC publication never moves DockerHub `latest`.
- Stable publication accepts only `vX.Y.Z` at the exact triggering commit **and current remote main tip**; it creates image tags `X.Y.Z` (plus architecture tags) and `latest`. Both channels require an existing tag, including annotated tags resolved to their commit. Each channel's publication is serialized without cancellation; live branch and tag checks occur before builds, source publication, registry login and version/channel updates. GitHub Releases are created with `--latest=false`; DockerHub `latest` is a separate stable image channel.
- Historical tags (including rc1/rc2 and `v1.0.0`) are not moved, deleted or rewritten. Legacy `preview-<full commit>` and other prerelease spellings do not authorize new publication. The source identity validator still accepts exact historical `preview-<full commit>` locators for existing downloads/rebuilds, and synthetic `test-<12 commit chars>` locators for local fixtures.
- Every source asset is named `mediagrap-source-<version>-<full commit>-linux-<arch>.tar.gz` under the explicit Release tag; no mutable latest-download URL is used. Archives are architecture-specific because ffprobe materials differ.
- Before any Docker login/push, `publish-source.py` creates or checks the durable public Release, uploads missing assets without overwrite, downloads both via unauthenticated HTTPS and verifies exact SHA, identity, inventory and legal materials. Existing asset mismatches fail closed; upload races fail rather than clobber. `publish-images.py` rejects mismatched existing immutable image tags, rechecks branch eligibility, and moves the mutable channel only after matching images exist. A branch can still move immediately after the last remote check; this is not a cross-service atomic transaction.

Actions artifacts retain source, manifests, diagnostics, verification, rebuild evidence and publication candidates for 30 days. They are intermediate evidence, **not the durable source offer**. Keep the public Release assets while distributing the corresponding images/binaries, including previews after `preview` moves. Do not delete or replace them to resume a failed publication. The application depends on GitHub's public availability for downloads: monitor the source link, retain independent source copies, and never distribute a candidate whose public source verification has not passed. `verify-image.py` without `--source-directory` checks the actual anonymous public download; local fixture success does not establish hosted availability.

These files implement publication gates, not proof of an executed release or native hosted runner allocation. Local builds and test-only snapshots are not distributable releases; their synthetic locators are exercised with owned fixtures and are not uploaded. Repository protections, live publication and release acceptance remain owner actions.

## Manual short-tag release

After this workflow is reviewed and merged onto the relevant branch through its required checks and approval, an authorized maintainer may explicitly publish a new RC as follows. These commands **create and push a tag, which triggers publication**; they are not part of local verification. Choose an unused version; do not reuse `v1.0.0` or move any existing tag.

```sh
# Run in a shell that stops on errors; no force/tag replacement.
set -eu
# RC example: replace with the next owner-approved unused version.
TAG=v1.0.1-rc.1
BRANCH=dev
# For stable instead, use TAG=v1.0.1 and BRANCH=main.
test -z "$(git status --porcelain)"
git fetch origin "$BRANCH" --tags
COMMIT=$(git rev-parse "refs/remotes/origin/$BRANCH^{commit}")
# Review this exact commit and its checks before continuing.
git show --no-patch --format=fuller "$COMMIT"
git tag -a "$TAG" "$COMMIT" -m "$TAG"
git push origin "refs/tags/$TAG"
```

No full SHA suffix is needed in the Git or user-facing image tag. The full commit remains in build metadata, source filenames, manifests, candidate proofs and baked `/source` URLs. A branch advancing (or a tag moving/disappearing) makes a candidate stale and stops later publication gates; use a new approved RC number at the new dev tip rather than force-updating tags. Source/assets or immutable architecture images may already exist after a partial run; keep them and retry only if the same exact identity is still eligible. No rollback or cross-service atomicity is promised. There is no `workflow_dispatch` release path and no automatic dev-push preview publication.

## Release acceptance

Before enabling tag publication, independently review both architecture evidence and perform a clean rebuild from the downloaded source archive, including a modified/relinked ffprobe check. This implementation stage is not a legal certification, historical relicensing, asset provenance audit or provider-terms review. The first-admin bootstrap behavior remains unchanged: isolate setup before untrusted access; see [deployment guidance](deployment.md#first-setup-and-tls). Secure cookies alone do not prevent first-visitor takeover.
