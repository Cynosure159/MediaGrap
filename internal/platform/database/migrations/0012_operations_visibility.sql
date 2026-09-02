CREATE TABLE IF NOT EXISTS job_events (
    id INTEGER PRIMARY KEY,
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    progress_current INTEGER NOT NULL DEFAULT 0,
    progress_total INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_events_cursor ON job_events(id);
CREATE INDEX IF NOT EXISTS idx_job_events_job ON job_events(job_id, id);

ALTER TABLE audit_entries ADD COLUMN outcome TEXT NOT NULL DEFAULT 'recorded';
ALTER TABLE audit_entries ADD COLUMN backup_path TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_entries ADD COLUMN recoverability TEXT NOT NULL DEFAULT 'not_applicable';
