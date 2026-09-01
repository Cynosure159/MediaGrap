ALTER TABLE artwork_plan_assets RENAME TO artwork_plan_assets_legacy;
ALTER TABLE artwork_plans RENAME TO artwork_plans_legacy;

CREATE TABLE artwork_plans (
    id TEXT PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK(state IN ('previewed', 'queued', 'running', 'applied', 'conflicted', 'failed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TEXT
);

CREATE TABLE artwork_plan_assets (
    id INTEGER PRIMARY KEY,
    artwork_plan_id TEXT NOT NULL REFERENCES artwork_plans(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('poster', 'fanart', 'clearlogo', 'clearart', 'discart', 'banner', 'landscape')),
    candidate_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    provider_asset_id TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL,
    preview_url TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT '',
    likes INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL,
    UNIQUE(artwork_plan_id, kind)
);

INSERT INTO artwork_plans(id, media_item_id, state, created_at, applied_at)
SELECT id, media_item_id, state, created_at, applied_at FROM artwork_plans_legacy;
INSERT INTO artwork_plan_assets(artwork_plan_id, kind, source_url, target_path)
SELECT artwork_plan_id, kind, source_url, target_path FROM artwork_plan_assets_legacy;
DROP TABLE artwork_plan_assets_legacy;
DROP TABLE artwork_plans_legacy;

CREATE TABLE artwork_candidates (
    id TEXT PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_asset_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('poster', 'fanart', 'clearlogo', 'clearart', 'discart', 'banner', 'landscape')),
    source_url TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    likes INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_item_id, provider, provider_asset_id, kind)
);
CREATE INDEX idx_artwork_candidates_media_kind ON artwork_candidates(media_item_id, kind, likes DESC);

CREATE TABLE artwork_assets (
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    provider_asset_id TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(media_item_id, kind)
);
