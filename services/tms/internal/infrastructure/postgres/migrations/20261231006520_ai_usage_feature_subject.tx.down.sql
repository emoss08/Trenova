DROP INDEX IF EXISTS "idx_ai_usage_records_tenant_feature_time";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_pair";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_type";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "subject_id";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "subject_type";

--bun:split
ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "feature";
