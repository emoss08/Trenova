DROP INDEX IF EXISTS "idx_ai_usage_records_tenant_failures";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "error_message";
