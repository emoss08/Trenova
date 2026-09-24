-- ai_logs recorded which feature made a model call and what it was about, but
-- nothing ever read it, and it kept previews of document text indefinitely.
-- Usage records already hold every call's provider, tokens, cost and latency;
-- they now also name the feature and the record the call was for, so ai_logs
-- can be retired without losing the one thing it said that usage did not.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "feature" varchar(50);

--bun:split
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "subject_type" varchar(50);

--bun:split
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "subject_id" varchar(100);

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract'));

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_type";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_subject_type" CHECK ("subject_type" IN ('Document', 'FormulaSchema'));

--bun:split
-- A subject is a type and an id together; either alone names nothing.
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_pair";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_subject_pair" CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL));

--bun:split
-- The by-feature breakdown and any one feature's history read this.
CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_feature_time"
    ON "ai_usage_records"("organization_id", "business_unit_id", "feature", "created_at" DESC);

COMMENT ON COLUMN "ai_usage_records"."feature" IS 'The product feature that made the call; null for calls made before features were recorded or by callers that name none';

COMMENT ON COLUMN "ai_usage_records"."subject_type" IS 'Kind of record the call was about, when it was about one';

COMMENT ON COLUMN "ai_usage_records"."subject_id" IS 'Id of the record the call was about; set together with subject_type';
