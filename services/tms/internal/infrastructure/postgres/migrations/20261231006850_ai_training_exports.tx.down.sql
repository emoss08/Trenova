DROP INDEX IF EXISTS "idx_ai_training_export_records_tenant";

--bun:split
DROP TABLE IF EXISTS "ai_training_export_records";

--bun:split
DROP INDEX IF EXISTS "uq_ai_training_exports_active";

--bun:split
DROP INDEX IF EXISTS "idx_ai_training_exports_created";

--bun:split
DROP TABLE IF EXISTS "ai_training_exports";
