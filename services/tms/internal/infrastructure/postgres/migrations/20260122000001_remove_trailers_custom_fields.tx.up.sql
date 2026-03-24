ALTER TABLE trailers DROP CONSTRAINT IF EXISTS chk_trailers_custom_fields;

--bun:split
DROP INDEX IF EXISTS idx_trailers_custom_fields;

--bun:split
ALTER TABLE trailers DROP COLUMN IF EXISTS custom_fields;
