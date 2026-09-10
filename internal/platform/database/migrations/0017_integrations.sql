ALTER TABLE jobs ADD COLUMN event_version INTEGER NOT NULL DEFAULT 0;
CREATE TABLE system_events (
 id TEXT PRIMARY KEY, event_type TEXT NOT NULL, aggregate_id TEXT NOT NULL,
 aggregate_version INTEGER NOT NULL, source_id INTEGER, payload_bytes BLOB NOT NULL,
 occurred_at INTEGER NOT NULL, dispatched_at INTEGER,
 UNIQUE(event_type, aggregate_id, aggregate_version)
);
CREATE INDEX system_events_pending ON system_events(dispatched_at, occurred_at);
CREATE TABLE webhook_endpoints (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, url TEXT NOT NULL, enabled INTEGER NOT NULL,
 event_types_json TEXT NOT NULL, source_ids_json TEXT NOT NULL, config_version INTEGER NOT NULL DEFAULT 1,
 active_key_id TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, deleted_at INTEGER
);
CREATE TABLE webhook_signing_keys (
 id TEXT PRIMARY KEY, endpoint_id TEXT NOT NULL REFERENCES webhook_endpoints(id),
 encrypted_secret BLOB NOT NULL, encryption_key_version INTEGER NOT NULL DEFAULT 1,
 created_at INTEGER NOT NULL, retired_at INTEGER
);
CREATE TABLE webhook_deliveries (
 id TEXT PRIMARY KEY, endpoint_id TEXT NOT NULL REFERENCES webhook_endpoints(id), event_id TEXT NOT NULL REFERENCES system_events(id),
 target_url TEXT NOT NULL, config_version INTEGER NOT NULL, state TEXT NOT NULL CHECK(state IN ('queued','delivering','retry_wait','succeeded','dead','cancelled')),
 attempt_count INTEGER NOT NULL DEFAULT 0, generation_attempts INTEGER NOT NULL DEFAULT 0, retry_generation INTEGER NOT NULL DEFAULT 0,
 next_attempt_at INTEGER NOT NULL, lease_owner TEXT, lease_expires_at INTEGER,
 last_status INTEGER NOT NULL DEFAULT 0, last_error_code TEXT NOT NULL DEFAULT '',
 created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, delivered_at INTEGER,
 UNIQUE(endpoint_id,event_id)
);
CREATE INDEX webhook_delivery_due ON webhook_deliveries(state,next_attempt_at);
CREATE INDEX webhook_delivery_lease ON webhook_deliveries(state,lease_expires_at);
CREATE INDEX webhook_delivery_history ON webhook_deliveries(endpoint_id,created_at,id);
CREATE TABLE webhook_attempts (
 delivery_id TEXT NOT NULL REFERENCES webhook_deliveries(id) ON DELETE CASCADE,
 attempt_number INTEGER NOT NULL, key_id TEXT NOT NULL, started_at INTEGER NOT NULL,
 duration_ms INTEGER, status_code INTEGER, error_code TEXT, PRIMARY KEY(delivery_id,attempt_number)
);
CREATE TABLE api_tokens (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, owner_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_prefix TEXT NOT NULL, token_hash BLOB NOT NULL UNIQUE, audience TEXT NOT NULL DEFAULT 'mcp',
 scopes_json TEXT NOT NULL, source_ids_json TEXT NOT NULL, expires_at INTEGER NOT NULL,
 last_used_at INTEGER, revoked_at INTEGER, created_at INTEGER NOT NULL
);
CREATE TABLE mcp_audit (
 id INTEGER PRIMARY KEY, token_id TEXT NOT NULL, owner_user_id INTEGER NOT NULL,
 method TEXT NOT NULL, result_code TEXT NOT NULL, request_id TEXT NOT NULL,
 duration_ms INTEGER NOT NULL, created_at INTEGER NOT NULL
);
CREATE TABLE api_token_sources (
 token_id TEXT NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
 source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
 PRIMARY KEY(token_id,source_id)
);
CREATE TABLE webhook_endpoint_sources (
 endpoint_id TEXT NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
 source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
 PRIMARY KEY(endpoint_id,source_id)
);
