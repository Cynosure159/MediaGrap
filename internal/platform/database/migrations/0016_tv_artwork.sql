CREATE TABLE tv_artwork_candidates (
    id TEXT PRIMARY KEY,
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK(scope IN ('show', 'season')),
    season_number INTEGER NOT NULL DEFAULT -1,
    provider TEXT NOT NULL,
    provider_asset_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('poster', 'fanart', 'clearlogo', 'logo', 'clearart', 'banner', 'landscape', 'character', 'season_poster', 'season_banner', 'season_landscape')),
    source_url TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    likes INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK((scope = 'show' AND season_number = -1) OR (scope = 'season' AND season_number >= 0)),
    UNIQUE(show_id, scope, season_number, provider, provider_asset_id, kind)
);

ALTER TABLE audit_entries ADD COLUMN show_id INTEGER REFERENCES tv_shows(id) ON DELETE SET NULL;
CREATE INDEX idx_audit_entries_show ON audit_entries(show_id, created_at DESC);

CREATE INDEX idx_tv_artwork_candidates_show_scope_kind ON tv_artwork_candidates(show_id, scope, season_number, kind, sort_order);

CREATE TABLE tv_artwork_plans (
    id TEXT PRIMARY KEY,
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK(state IN ('previewed', 'queued', 'running', 'applied', 'conflicted', 'failed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TEXT
);

CREATE TABLE tv_artwork_plan_assets (
    id INTEGER PRIMARY KEY,
    artwork_plan_id TEXT NOT NULL REFERENCES tv_artwork_plans(id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK(scope IN ('show', 'season')),
    season_number INTEGER NOT NULL DEFAULT -1,
    kind TEXT NOT NULL CHECK(kind IN ('poster', 'fanart', 'clearlogo', 'logo', 'clearart', 'banner', 'landscape', 'character', 'season_poster', 'season_banner', 'season_landscape')),
    candidate_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_asset_id TEXT NOT NULL,
    source_url TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    likes INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL,
    CHECK((scope = 'show' AND season_number = -1) OR (scope = 'season' AND season_number >= 0)),
    UNIQUE(artwork_plan_id, scope, season_number, kind)
);

CREATE TABLE tv_artwork_assets (
    show_id INTEGER NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    scope TEXT NOT NULL CHECK(scope IN ('show', 'season')),
    season_number INTEGER NOT NULL DEFAULT -1,
    kind TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_asset_id TEXT NOT NULL,
    target_path TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(show_id, scope, season_number, kind),
    CHECK((scope = 'show' AND season_number = -1) OR (scope = 'season' AND season_number >= 0))
);
