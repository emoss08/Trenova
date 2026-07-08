ALTER TABLE "edi_templates"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'C')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_templates_search"
    ON "edi_templates" USING GIN("search_vector");

--bun:split
ALTER TABLE "edi_mapping_profiles"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_mapping_profiles_search"
    ON "edi_mapping_profiles" USING GIN("search_vector");
