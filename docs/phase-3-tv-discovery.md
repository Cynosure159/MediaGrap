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

- GET /api/v1/tv/shows?q= returns indexed TV shows with season and episode counts.
- GET /api/v1/tv/shows/{id} returns the read-only episode list.

The library now has separate Movie and TV shows tabs. Desktop retains the catalog/detail layout; on mobile, selecting a show replaces the catalog until the user closes the detail panel.

## Deferred Phase 3 work

- TMDb TV matching and details, including the configured metadata language.
- Kodi show, season, and episode NFO parse/write previews.
- Season/episode artwork downloads.
- Missing-episode reporting, batch selection, persisted scrape jobs, cancellation, and retry-only-failed behavior.
