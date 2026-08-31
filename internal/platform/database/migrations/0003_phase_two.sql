CREATE TABLE IF NOT EXISTS media_metadata (
    media_item_id INTEGER PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    provider_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    original_title TEXT NOT NULL DEFAULT '',
    year INTEGER,
    overview TEXT NOT NULL DEFAULT '',
    runtime_minutes INTEGER,
    genres_json TEXT NOT NULL DEFAULT '[]',
    poster_url TEXT NOT NULL DEFAULT '',
    backdrop_url TEXT NOT NULL DEFAULT '',
    locked_fields_json TEXT NOT NULL DEFAULT '[]',
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS write_plans (
    id TEXT PRIMARY KEY,
    media_item_id INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    target_path TEXT NOT NULL,
    content TEXT NOT NULL,
    state TEXT NOT NULL CHECK(state IN ('previewed', 'applied', 'conflicted', 'failed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TEXT
);

CREATE TABLE IF NOT EXISTS audit_entries (
    id INTEGER PRIMARY KEY,
    action TEXT NOT NULL,
    media_item_id INTEGER REFERENCES media_items(id) ON DELETE SET NULL,
    target_path TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_write_plans_media_item ON write_plans(media_item_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entries_media_item ON audit_entries(media_item_id, created_at DESC);
