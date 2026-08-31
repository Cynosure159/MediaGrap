ALTER TABLE media_metadata ADD COLUMN rating REAL;
ALTER TABLE media_metadata ADD COLUMN votes INTEGER;
ALTER TABLE media_metadata ADD COLUMN content_rating TEXT NOT NULL DEFAULT '';
ALTER TABLE media_metadata ADD COLUMN directors_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE media_metadata ADD COLUMN writers_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE media_metadata ADD COLUMN studios_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE media_metadata ADD COLUMN cast_json TEXT NOT NULL DEFAULT '[]';
