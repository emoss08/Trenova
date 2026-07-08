DROP INDEX IF EXISTS "idx_edi_templates_search";

--bun:split
ALTER TABLE "edi_templates"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
DROP INDEX IF EXISTS "idx_edi_mapping_profiles_search";

--bun:split
ALTER TABLE "edi_mapping_profiles"
    DROP COLUMN IF EXISTS "search_vector";
