CREATE TABLE IF NOT EXISTS artwork_plans (
    id TEXT PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK(state IN ('previewed', 'applied', 'conflicted', 'failed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TEXT
);

CREATE TABLE IF NOT EXISTS artwork_plan_assets (
    id INTEGER PRIMARY KEY,
    artwork_plan_id TEXT NOT NULL REFERENCES artwork_plans(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('poster', 'fanart')),
    source_url TEXT NOT NULL,
    target_path TEXT NOT NULL,
    UNIQUE(artwork_plan_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_artwork_plans_media_item ON artwork_plans(media_item_id, created_at DESC);
