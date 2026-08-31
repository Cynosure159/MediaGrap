CREATE TABLE IF NOT EXISTS application_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO application_meta(key, value) VALUES ('created_at', CURRENT_TIMESTAMP);

