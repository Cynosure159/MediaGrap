# Roadmap

[Current usage](usage.md) · [Architecture](architecture.md) · [Release acceptance](distribution.md#release-acceptance)

This roadmap separates current code from intended work, without phase numbers or prototype feature promises. Priorities: **data safety → bounded resource use → usable desktop/mobile workflows → extensibility/observability → breadth**. No dates or release versions are promised.

## Current baseline

The current tree provides movie/TV discovery, bounded incremental/full scans, local NFO hydration with saved-metadata priority, TMDb matching, Fanart.tv artwork, Kodi NFO writes, ffprobe/file audit, browser movie/TV rename plans, persisted jobs/SSE/schedules, proxies, bilingual/theme preferences, Webhooks and scoped MCP automation with browser-approved movie file plans.

The open-source preparation adds current-project AGPL-3.0-only licensing, matching-build `/source` delivery and bundled dependency/legal/build materials, explicit secure-cookie configuration, and main/dev plus dual-architecture tag CI. These are **not claims about the already-published 0.0.6 image**. Actual publication, clean release tags, hosting/account configuration and independent acceptance remain separate owner actions.

## Highest priority

- **First-administrator takeover mitigation — explicitly deferred implementation.** Require one-time bootstrap proof or a deliberately trusted setup-network design before recommending unattended public first boot. Design credential provisioning/expiry, concurrent setup rejection, safe recovery and deployment migration; add rejection and recovery tests. The current first-visitor behavior/default binding remains unchanged. Isolated setup is required now; Secure cookies do not solve this risk.
- Establish a verified private vulnerability-reporting channel and supported-release policy; configure repository protections/CI secrets only through authorized owner actions. Review distribution evidence and regenerate archives from the final reviewed tree before any release.
- Backup/restore with explicit plan/apply, retention and recoverable scope; upgrade/downgrade guidance and pre-migration backup design. Today operators must stop and back up complete config/WAL/key state themselves.
- Broaden filesystem recovery guarantees only after fault-injection and target NAS validation. Browser TV batches/directory renames do not inherit automation movie recovery. No global rollback claim; keep partial results visible.

## Next workflow improvements

- TV missing-episode reporting against explicitly matched provider inventories, distinguishing absent files, unmatched shows and specials; complete multi-episode NFO semantics.
- Persisted parent/child TV scrape/write batches, scoped progress/cancellation and retry-only-failed children. Existing synchronous browser matching can write several NFOs and must remain clearly labeled until deliberately changed.
- Field-by-field metadata comparison/merge/locks in the UI; controlled raw XML editing/formatting/validation and unknown-field compatibility policies. Do not market NFO preview as a completed editor.
- Artwork uploads/local selection/crop only with format/size/path/replacement safety; richer cast editing/order and bounded portrait downloads.
- Conditional naming templates and remaining batch ergonomics without weakening collision, preview or permission checks. Movie and TV rename execution already exists; do not list all renaming as unimplemented.

## Operations and integrations

- Measure large-library behavior on actual NAS/NFS/SMB, including cancellation latency, lock waits, disk saturation and peak memory; local synthetic benchmark speedups are not production guarantees. Preserve two-reader/200-candidate bounds and SQLite durability unless a separately reviewed change is justified.
- Extend completion observability beyond the bounded recent-history window only with a concrete design. Preserve drafts, missing-target gating and selection-generation safety as catalogs evolve.
- Duplicate reports, additional metadata/artwork/ID filters, CSV export and explicit-scope Kodi JSON-RPC synchronization.
- MCP OAuth interoperability, additional queries/resources, subscriptions or stdio only for validated client needs. No automatic file self-approval; any unattended write policy needs separate authorization design.
- Review supply-chain pinning, dependency/image scanning and signing as release operations. Existing CI is not a claim of signatures, fully offline rebuilds or legal certification.

## Later, demand-driven

Concert/music sidecar metadata (not ID3 rewriting), more official providers, HTML export, multi-user roles, a PostgreSQL adapter and external workers. These are not current supported media types/deployment options. Playback, transcoding, media acquisition and arbitrary in-process plugins remain outside the product's focus.

## Acceptance for every change

Include backend/API/migration tests, desktop/mobile behavior, English/Chinese messages, safe diagnostic logs and the corresponding maintained guide. Filesystem features require preview/revalidation/conflict/partial-failure tests using owned fixtures; integrations require redaction and policy tests. Report unverified filesystem/client/release boundaries honestly. Preserve runtime branding, theme tokens, reusable components and mandatory upstream notices; exported prototypes are not an implementation dependency.
