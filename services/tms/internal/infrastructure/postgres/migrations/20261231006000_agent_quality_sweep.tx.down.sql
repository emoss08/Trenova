DROP INDEX IF EXISTS "idx_ai_usage_records_evaluation";

--bun:split
ALTER TABLE "assistant_turns" DROP COLUMN IF EXISTS "fingerprint";

--bun:split
ALTER TABLE "agent_runs" DROP COLUMN IF EXISTS "fingerprint";

--bun:split
DROP INDEX IF EXISTS "uq_agent_evaluations_suite_ordinal";

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_suite_ordinal";

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "fk_agent_evaluations_suite_run";

--bun:split
DELETE FROM "agent_evaluations" WHERE "suite_run_id" IS NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "suite_ordinal";

ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "suite_run_id";

--bun:split
DROP TABLE IF EXISTS "agent_quality_controls";

--bun:split
DROP TABLE IF EXISTS "agent_suite_runs";
