DROP INDEX IF EXISTS "idx_ai_corrections_document";

--bun:split
DROP INDEX IF EXISTS "idx_extraction_shadow_results_scored";

--bun:split
DROP INDEX IF EXISTS "idx_extraction_shadow_results_document";

--bun:split
DROP INDEX IF EXISTS "idx_extraction_shadow_results_created";

--bun:split
DROP TABLE IF EXISTS "extraction_shadow_results";

--bun:split
DROP TABLE IF EXISTS "extraction_shadow_settings";
