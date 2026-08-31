CREATE TABLE tv_metadata (
    show_id INTEGER PRIMARY KEY REFERENCES tv_shows(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    provider_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    original_title TEXT NOT NULL DEFAULT '',
    year INTEGER,
    overview TEXT NOT NULL DEFAULT '',
    genres_json TEXT NOT NULL DEFAULT '[]',
    poster_url TEXT NOT NULL DEFAULT '',
    backdrop_url TEXT NOT NULL DEFAULT '',
    rating REAL,
    votes INTEGER,
    status TEXT NOT NULL DEFAULT '',
    network TEXT NOT NULL DEFAULT '',
    cast_json TEXT NOT NULL DEFAULT '[]',
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tv_episode_metadata (
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL CHECK(season_number >= 0),
    episode_number INTEGER NOT NULL CHECK(episode_number >= 0),
    title TEXT NOT NULL DEFAULT '',
    overview TEXT NOT NULL DEFAULT '',
    air_date TEXT NOT NULL DEFAULT '',
    runtime_minutes INTEGER,
    still_url TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(show_id, season_number, episode_number)
);

CREATE INDEX tv_episode_metadata_show_season ON tv_episode_metadata(show_id, season_number, episode_number);
