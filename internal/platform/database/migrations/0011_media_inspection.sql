CREATE TABLE media_probe_cache (
    media_item_id INTEGER PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
    file_size INTEGER NOT NULL,
    modified_at TEXT NOT NULL,
    probe_json TEXT NOT NULL,
    probed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

