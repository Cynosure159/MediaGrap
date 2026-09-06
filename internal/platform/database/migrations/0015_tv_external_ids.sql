ALTER TABLE tv_metadata ADD COLUMN tvdb_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tv_metadata ADD COLUMN imdb_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_tv_metadata_tvdb_id ON tv_metadata(tvdb_id) WHERE tvdb_id <> '';
