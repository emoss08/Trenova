ALTER TABLE trailers ADD COLUMN IF NOT EXISTS custom_fields jsonb DEFAULT '{}'::jsonb;

--bun:split
ALTER TABLE trailers ADD CONSTRAINT chk_trailers_custom_fields CHECK (jsonb_typeof(custom_fields) = 'object');

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_custom_fields ON trailers USING GIN(custom_fields);
