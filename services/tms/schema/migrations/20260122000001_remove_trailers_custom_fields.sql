ALTER TABLE trailers DROP CONSTRAINT IF EXISTS chk_trailers_custom_fields;

DROP INDEX IF EXISTS idx_trailers_custom_fields;

ALTER TABLE trailers DROP COLUMN IF EXISTS custom_fields;
