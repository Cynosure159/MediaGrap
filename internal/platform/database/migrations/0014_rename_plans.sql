CREATE TABLE rename_plans (
    id TEXT PRIMARY KEY,
    media_item_id INTEGER REFERENCES media_items(id) ON DELETE SET NULL,
    tv_show_id INTEGER REFERENCES tv_shows(id) ON DELETE SET NULL,
    pattern TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'previewed',
    items_json TEXT NOT NULL DEFAULT '[]',
    warnings_json TEXT NOT NULL DEFAULT '[]',
    has_conflicts INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TEXT,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
