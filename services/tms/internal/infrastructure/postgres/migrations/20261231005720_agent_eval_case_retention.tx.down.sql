DROP INDEX IF EXISTS "idx_agent_eval_cases_retention";

--bun:split
ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_agent_eval_case_period";

--bun:split
ALTER TABLE "data_retention" DROP COLUMN IF EXISTS "agent_eval_case_retention_period";
