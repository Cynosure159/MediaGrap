ALTER TABLE sources ADD COLUMN scan_mode TEXT NOT NULL DEFAULT 'incremental' CHECK(scan_mode IN ('full', 'incremental'));
ALTER TABLE sources ADD COLUMN schedule_enabled INTEGER NOT NULL DEFAULT 0 CHECK(schedule_enabled IN (0, 1));
ALTER TABLE sources ADD COLUMN schedule_interval_minutes INTEGER NOT NULL DEFAULT 1440 CHECK(schedule_interval_minutes BETWEEN 15 AND 10080);
ALTER TABLE sources ADD COLUMN next_scan_at TEXT;
ALTER TABLE sources ADD COLUMN last_scan_at TEXT;

CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme TEXT NOT NULL DEFAULT 'dark' CHECK(theme IN ('dark', 'light', 'system')),
    locale TEXT NOT NULL DEFAULT 'en' CHECK(locale IN ('en', 'zh-CN')),
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE connection_test_results (
    id INTEGER PRIMARY KEY,
    target TEXT NOT NULL CHECK(target IN ('tmdb', 'fanart_tv', 'proxy')),
    status TEXT NOT NULL CHECK(status IN ('reachable', 'failed', 'not_configured')),
    http_status INTEGER,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    tested_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_connection_test_results_target_id ON connection_test_results(target, id DESC);
