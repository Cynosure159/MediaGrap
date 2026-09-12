# Licensing and bundled materials

The owner selected **AGPL-3.0-only** for the current MediaGrap project. See the complete [project license](../../LICENSE). This does not relicense third-party works, provider content, or every historic commit/image. Preserve upstream copyright and license notices. The already-published `cynosure159/mediagrap:0.0.6` predates this change; neither this license selection nor the new source mechanism is a claim about that image.

A distribution build embeds a same-build source archive, publicly downloadable at `/source` without authentication. Missing/mismatched development archives return HTTP 503, not a guessed repository URL. The archive contains the reviewed tracked project tree, dependency materials, manifests, checksums and build recipe. This is not a blanket legal certification.

## Actual build inventory

- Go: entire exact module zip sources, including all nested LICENSE/COPYING files; MCP's complete mixed MIT/Apache transition text is preserved, not replaced by one SPDX label. Go builder runtime LICENSE and toolchain version are included.
- Frontend: Vite records nonempty rendered modules. The inspected production bundle retains Vue shared/reactivity/runtime-core/runtime-dom 3.5.42, vue-router 4.6.4 and workbox-window 7.4.1. Whole npm distributions are preserved as build inputs, not misidentified as preferred upstream source. Although `@vue/devtools-api` is absent from rendered application output, the shipped vue-router global bundle contains its 6.6.4 implementation. Its [exact upstream MIT notice](vue-devtools-api-6.6.4-LICENSE) is supplied separately without changing the npm distribution or synthesizing attribution. The version-tagged upstream `packages/api/package.json` was verified against the installed package. Vue 3.5.42 and router 4.6.4 preferred TypeScript source trees, lockfiles and build configurations are supplied as checksum-pinned upstream archives in `third-party/frontend-upstream`, alongside the nested-code notice and origin/package [inventory](frontend-sources.json). Packaging validates archive checksums, source package versions and required build files. All exact Workbox packages are included because service-worker generation runs separately; their packages retain TypeScript source and notices. Not every dev/test tool is vendored. See [dependency build instructions](../distribution.md#vue-and-router-preferred-source). Rebuilding requires registry/toolchain availability; offline reconstruction has not been established.
- FFmpeg 7.1.1: exact unchanged archive, all COPYING texts and LICENSE.md, unchanged minimal static configure flags, generated configuration/logs, actual `ffprobe -L/-version/-buildconf`, and linker map. Default configuration selects LGPL-2.1-or-later, not GPL/nonfree. Full source and the build script permit rebuilding/replacing the standalone ffprobe, including modified library code; merely naming an upstream URL is not the distribution mechanism.
- Static musl 1.2.5 (including Alpine musl-dev-owned libssp_nonshared.a; map distinguishes searched archives from selected members): upstream archive and complete COPYRIGHT (MIT plus upstream subcomponent statements). Builder refuses an unreviewed musl version. Alpine package revisions/compiler/linker versions are recorded; upstream musl source is not asserted to be Alpine's patched source. MIT requires notice preservation, not publication of every downstream patch.
- GCC 14.2.0 support: actual static linkage recorded in link.map. Complete GPLv3 and GCC Runtime Library Exception 3.1 are retained. The exception applies to eligible compilation; this normal GCC build does not load a proprietary compiler plugin. Build tools themselves are not shipped in scratch. Do not incorrectly apply the compiler's GPL to all generated executables or claim all toolchain source is mandatory under the runtime exception.
- Mozilla CA data: exact Alpine ca-certificates 20260611 source archive (including certdata.txt under MPL-2.0 and upstream script notices) and full MPL text. Installed CA package version is pinned; runtime bundle checksum is recorded. APKBUILD source SHA512 independently matches the retrieved source. The CA bundle is separate from the application's AGPL license.

## Pinned retrieval evidence

These SHA256 values were measured from official HTTPS downloads (not release-signature attestations):

| Material | SHA256 |
| --- | --- |
| vuejs/core v3.5.42 source archive | `88c4dbb31890fe252a8694887ea91fbf7b84f5c524b10e6f01ce4dbf6b19a7e6` |
| vuejs/router v4.6.4 source archive | `99024e4f23a85e9bb4af68fbebbd8f857c88156a6073cafd71d75211ef5eabfd` |
| vuejs/devtools-v6 v6.6.4 LICENSE | `050bbca6960784db52ff387271bf2ecc5cbed7cf8581b415d528a6ecb6585015` |
| GNU agpl-3.0.txt | `0d96a4ff68ad6d4b6f1f30f713b18d5184912ba8dd389f86aa7710db079abcb0` |
| ffmpeg.org/releases/ffmpeg-7.1.1.tar.xz | `733984395e0dbbe5c046abda2dc49a5544e7e0e1e2366bba849222ae9e3a03b1` |
| musl.libc.org/releases/musl-1.2.5.tar.gz | `a9a118bbe84d8764da0ea0d28b3ab3fae8477fc7e4085d90102b8596fc7c75e4` |
| Alpine ca-certificates/-/archive/20260611/ca-certificates-20260611.tar.bz2 | `32ca73f2e81e2b88dc614f12e1ee04a82b1ec5a8e29d9f359ddf8905a0afcbb0` |
| gcc-mirror/gcc releases/gcc-14.2.0 COPYING.RUNTIME | `9d6b43ce4d8de0c878bf16b54d8e7a10d9bd42b75178153e3af6a815bdc90f74` |
| gcc-mirror/gcc releases/gcc-14.2.0 COPYING3 | `8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903` |
| mozilla.org/media/MPL/2.0/index.815ca599c9df.txt | `fab3dd6bdab226f1c08630b1dd917e11fcb4ec5e1e020e2c16f83a0a13863e85` |

Source publication and binary/image redistribution are separate events. Provider service terms and asset provenance are outside this implementation; no provenance investigation or historic relicensing is claimed.
