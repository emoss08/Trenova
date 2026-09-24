ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "agent_eval_case_retention_period" integer NOT NULL DEFAULT 365;

COMMENT ON COLUMN "data_retention"."agent_eval_case_retention_period" IS 'Days an agent evaluation case is kept after it was captured before it is purged with its evaluations; 0 keeps cases until they expire or their conversation is deleted';

--bun:split
ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_agent_eval_case_period";

--bun:split
ALTER TABLE "data_retention"
    ADD CONSTRAINT "ck_data_retention_agent_eval_case_period" CHECK ("agent_eval_case_retention_period" >= 0);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_retention"
    ON "agent_eval_cases"("organization_id", "business_unit_id", "created_at");
