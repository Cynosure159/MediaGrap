CREATE TABLE tv_shows (
    id INTEGER PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL,
    title_hint TEXT NOT NULL,
    year_hint INTEGER,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, relative_path)
);
CREATE INDEX tv_shows_source_title ON tv_shows(source_id, title_hint COLLATE NOCASE);

CREATE TABLE tv_seasons (
    id INTEGER PRIMARY KEY,
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL CHECK(season_number >= 0),
    UNIQUE(show_id, season_number)
);

CREATE TABLE tv_episodes (
    media_item_id INTEGER PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    season_id INTEGER NOT NULL REFERENCES tv_seasons(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL CHECK(season_number >= 0),
    episode_start INTEGER NOT NULL CHECK(episode_start >= 0),
    episode_end INTEGER NOT NULL CHECK(episode_end >= episode_start),
    title_hint TEXT NOT NULL,
    UNIQUE(show_id, season_number, episode_start, media_item_id)
);
CREATE INDEX tv_episodes_show_season_episode ON tv_episodes(show_id, season_number, episode_start);
