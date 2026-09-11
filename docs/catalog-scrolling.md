# Movie catalog continuous scrolling

The movie catalog has no page-number or next-page controls. It shows the server's total for the current search/filter immediately, and loads more rows as the scroll viewport approaches the bottom. The footer distinguishes loaded rows from the true matching total. A failed batch keeps earlier results and offers a localized retry button.

Transport remains bounded: `/api/v1/media?page=&pageSize=50&q=&filter=&sort=` fetches 50 records at a time. No eager whole-library fetch is performed. A request already in flight prevents duplicate scroll requests. New searches, filters, and sorts abort the old request and use a generation guard to reject late responses. Subsequent batches are deduplicated by item ID; no-progress batches stop with a retry state rather than repeatedly requesting automatically. Loading stops at the server total. Mutable libraries can change between batches; refresh the catalog after scans/deletions or concurrent metadata edits when a consistent new ordering is needed.

Filtering and sorting operate on the server, not on just the loaded subset. `filter` accepts `all`, `unscraped`/`nfo_missing`, `poster_missing`, `4k`, and `1080p`; `sort` accepts `title`, `year`, and `size`. Unsupported values retain the default unfiltered/title order. NFO/image filters use the scan's sidecar index; resolution uses filename hints, not probing. Movie responses expose optional metadata `year` separately from the original filename `yearHint`. Row years and year sorting prefer saved metadata, falling back to hints only when metadata has no year (for example, Blade Runner 2049 displays its saved 2017 release year). Title/year/size orders include a stable ID tie-breaker. The existing UI currently exposes All, Unscraped, 4K and 1080p filters. Counts beside individual filter choices are omitted rather than presenting partial-page counts as global statistics.

The catalog renders only the viewport plus five overscan rows on either side. Rows follow the 56px catalog design with 4px spacing; spacer elements preserve scrolling height. Images use native lazy loading and asynchronous decoding and are unmounted with off-screen rows. Loaded lightweight records remain in memory, but DOM nodes and decoded images do not grow with the full library. Movie selection remains ID-based, independent of whether its row is currently mounted. Mobile uses the same scroll viewport above the native bottom navigation; the workspace continues to own responsive sizing.

## Verification

- Composable tests cover >50 records, correct total, on-demand append, concurrent-event suppression, last batch, stale search rejection, and retry without duplicates.
- Component tests cover virtualized DOM size, lazy image attributes, scroll loading, and retry states.
- Backend fixture tests cover 137 entries across batches, identical-title ordering, and global filter/sort totals.
- ego-browser QA used an isolated local server and 137 scanned temporary movie/NFO fixtures: desktop and 390×844 mobile scrolled through all records; the catalog fetched 50 → 100 → 137, retained around 16–21 DOM rows, and displayed `137 Movies`. Searching for item 137 returned one match and its inspector opened normally. Adding fixture posters confirmed only rendered thumbnails were requested, all marked lazy/async. Mobile had no horizontal overflow.

No production library or provider credentials are required for these tests. Screenshots are diagnostic artifacts, not bundled application assets.
