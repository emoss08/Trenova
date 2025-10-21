DROP TABLE IF EXISTS "ai_logs";

DROP TYPE IF EXISTS "operation_enum";

DROP TYPE IF EXISTS "model_enum";

DROP FUNCTION IF EXISTS prevent_ai_logs_modification();

DROP TRIGGER IF EXISTS enforce_ai_logs_append_only ON "ai_logs";

DROP INDEX IF EXISTS "idx_ai_logs_timestamp";

DROP INDEX IF EXISTS "idx_ai_logs_org_bu";

DROP INDEX IF EXISTS "idx_ai_logs_user";

