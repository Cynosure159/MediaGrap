# Frontend URL routing

MediaGrap uses browser history URLs so a refresh, bookmark, or shared link can
restore the current library location. The URL is the source of truth for the
primary page, the selected media unit, and the Inspector workshop tab.

## Routes

| URL | Meaning |
| --- | --- |
| `/movies` | Movie catalog with no selected item |
| `/movies?movie=123` | Movie `123`, Overview workshop |
| `/movies?movie=123&tab=artwork` | Movie `123`, Artwork workshop |
| `/shows` | TV catalog with no selected show |
| `/shows?show=20` | Show `20` |
| `/shows?show=20&season=2` | Season 2 of show `20` |
| `/shows?show=20&season=2&episode=456` | Episode `456` in season 2 of show `20` |
| `/shows?show=20&season=2&episode=456&tab=nfo` | Episode `456`, NFO workshop |
| `/jobs` | Background jobs and audit/operations view |
| `/settings` | Settings, including media source management |

The valid workshop values are `overview`, `artwork`, `cast`, `nfo`, and
`files`. `overview` is omitted from generated URLs. Invalid or incomplete
query parameters are removed and the URL is canonicalized. TV selection
specificity is episode, then season, then show.

The former `/sources` entry is retained as a redirect to `/settings` for old
bookmarks. Sources remain managed inside Settings but are no longer a primary
navigation item.

Changing a page, media selection, or workshop creates a browser history entry;
search and filter state can be added later without changing this contract.
