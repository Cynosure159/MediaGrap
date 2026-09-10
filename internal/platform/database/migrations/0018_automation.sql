ALTER TABLE jobs ADD COLUMN actor_token_id TEXT;
CREATE TABLE automation_requests (
 token_id TEXT NOT NULL REFERENCES api_tokens(id), tool TEXT NOT NULL, idempotency_key TEXT NOT NULL,
 request_hash TEXT NOT NULL, job_id INTEGER NOT NULL REFERENCES jobs(id), created_at INTEGER NOT NULL,
 PRIMARY KEY(token_id,tool,idempotency_key)
);
CREATE TABLE automation_results (job_id INTEGER PRIMARY KEY REFERENCES jobs(id), result_json TEXT NOT NULL);
CREATE TABLE automation_plans (
 id TEXT PRIMARY KEY, token_id TEXT NOT NULL REFERENCES api_tokens(id), source_id INTEGER NOT NULL,
 media_id INTEGER NOT NULL, kind TEXT NOT NULL, digest TEXT NOT NULL,
 operations_json TEXT NOT NULL, state TEXT NOT NULL, created_at INTEGER NOT NULL, expires_at INTEGER NOT NULL,
 approved_by INTEGER REFERENCES users(id), approved_at INTEGER, approval_expires_at INTEGER, job_id INTEGER REFERENCES jobs(id),
 root_identity TEXT NOT NULL, metadata_digest TEXT NOT NULL
);
CREATE TABLE automation_operations (
 plan_id TEXT NOT NULL REFERENCES automation_plans(id), operation_index INTEGER NOT NULL,
 state TEXT NOT NULL, expected_hash TEXT NOT NULL DEFAULT '', temporary_path TEXT NOT NULL DEFAULT '',
 error_code TEXT NOT NULL DEFAULT '', updated_at INTEGER NOT NULL, PRIMARY KEY(plan_id,operation_index)
);
CREATE INDEX automation_plans_pending ON automation_plans(state,created_at);
CREATE TABLE automation_plan_requests (plan_id TEXT PRIMARY KEY REFERENCES automation_plans(id), request_hash TEXT NOT NULL);
