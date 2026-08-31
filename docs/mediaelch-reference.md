# MediaElch reference review

Reviewed on 2026-08-31 from the upstream repository and project documentation.

## Sources

- Repository: <https://github.com/Komet/MediaElch>
- Documentation: <https://mediaelch.github.io/mediaelch-doc/>
- Settings overview: <https://mediaelch.github.io/mediaelch-doc/settings.html>
- Quick start and media layout: <https://mediaelch.github.io/mediaelch-doc/quick-start.html>
- Renaming behavior: <https://mediaelch.github.io/mediaelch-doc/renaming.html>
- Kodi synchronization: <https://mediaelch.github.io/mediaelch-doc/kodi.html>
- Music behavior: <https://mediaelch.github.io/mediaelch-doc/music/index.html>

## Useful concepts to retain

MediaElch's source tree separates data, database, file search, media models, media-center compatibility, networking, renaming, scrapers, settings, UI, and workers. That validates several boundaries proposed for MediaGrap, even though the implementation language and deployment model differ.

Functional concepts worth preserving:

- distinct workflows for movies, TV shows/episodes, concerts, and music;
- Kodi-compatible NFO sidecars and configurable artwork filenames;
- browsing, scraping, reviewing, editing, and explicitly saving metadata;
- multiple metadata/artwork sources with language preferences;
- folder scanning for common movie, TV, DVD, and Blu-ray layouts;
- template-based renaming with conditionals and a dry run;
- proxy configuration;
- Kodi synchronization and CSV/HTML export as later features;
- visibility into missing episodes, duplicate movies, missing IDs, and missing artwork.

The upstream documentation currently lists movie/TV/concert/music management, TMDb, IMDb, TVMaze and other scraper sources, Fanart.tv artwork, music sources, trailers/themes, proxy settings, Kodi connection settings, and export/import settings. Availability and permitted use of every upstream provider must be rechecked against its official API and terms before MediaGrap implements it.

## Changes for a server-first product

- Replace desktop dialogs and synchronous selection flows with durable jobs, resumable UI state, and browser-safe previews.
- Separate discovery from scraping so a filesystem scan never unexpectedly generates provider traffic.
- Add authentication, CSRF protection, path capability boundaries, and explicit reverse-proxy trust.
- Make all file operations safe under concurrent web sessions and container/NAS mount behavior.
- Optimize lists and thumbnails for remote/mobile access instead of assuming local desktop resources.
- Store job/audit state so container restarts are observable and recoverable.
- Prefer official, documented APIs. HTML scraping is fragile and may conflict with provider terms.

## Licensing boundary

MediaElch is distributed under LGPL-3.0. MediaGrap may study its public behavior, formats, and architecture, but should not copy source code, tests, artwork, or other protected assets unless the new project's license and compliance obligations have been deliberately chosen and documented.

Kodi NFO compatibility should be implemented from public Kodi documentation plus independently created fixtures. Provider adapters should use each provider's official documentation and comply with attribution, API-key, caching, and branding requirements.

## What is intentionally not copied

- Qt/C++ UI structure and desktop lifecycle.
- Provider-specific implementation details or HTML parsers.
- Upstream assets, icons, translations, export themes, and test fixtures.
- Assumptions that a user is present to resolve every prompt synchronously.

