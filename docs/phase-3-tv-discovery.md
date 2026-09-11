# Phase 3: TV discovery

This first Phase 3 increment extends the existing safe, read-only scanner to recognize episodic video files and display a TV-focused library view. It does not scrape TV metadata or write any TV NFO/artwork yet.

## Discovery model

Each video file remains a media_items record. When its filename has a recognized episode marker, the scanner additionally creates or updates:

- one tv_shows record per source and show directory;
- one tv_seasons record per show and numeric season;
- one tv_episodes record linked to the existing video item.

The movie list excludes records currently associated with an episode. A later scan can reclassify a file safely: if its filename no longer matches a TV pattern, only its derived TV association is removed; its indexed media file and sidecars remain untouched.

## Supported filename patterns

- Show.Name.S02E03.mkv
- Show Name - 1x02 - Pilot.mp4
- Show.S01E01E02.mkv for a file containing a consecutive pair of episodes

For a conventional Show Name/Season 01/... layout, the show title is derived from the directory above the season directory. Otherwise it is derived from the filename prefix before the episode marker. Year hints in a show directory such as Show Name (2024) are retained.

The scanner still skips symbolic links and only reads filesystem metadata. It does not use episode title guesses to overwrite any existing NFO data.

## APIs and UI

- GET /api/v1/tv/shows?q= returns indexed TV shows with season and episode counts. The optional `title` is the saved metadata title; `titleHint` remains the original filename/directory hint. Catalog rows, tooltips, and alphabetical sorting prefer `title`. Search matches both names and the indexed path. After scraping or saving metadata, the workspace refreshes the show list and expanded episode caches without requiring a rescan or page reload.
- GET /api/v1/tv/shows/{id} returns the read-only episode list.
- The TV catalog is an expandable show → season → episode tree. Selecting any
  level scopes the inspector to that show, season, or episode.

The library now has separate Movie and TV shows tabs. Desktop retains the catalog/detail layout; on mobile, selecting a show replaces the catalog until the user closes the detail panel.

## Implemented follow-up work

- TMDb show matching and Kodi show/season/episode NFO writes are available.
- After a show has a TMDb association, a season or a single episode can be
  scraped independently. Only its SQLite metadata and its corresponding NFO
  targets are replaced; artwork and video files are unchanged.

## Deferred Phase 3 work

- Episode-specific artwork downloads beyond the existing TMDb still image.
- Missing-episode reporting, batch selection, persisted scrape jobs, cancellation, and retry-only-failed behavior.

Show and season Fanart.tv artwork downloads are implemented through the TV
Artwork workshop and durable safe-write jobs.
