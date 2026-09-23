DROP INDEX IF EXISTS "idx_agent_evaluations_case";

--bun:split
DELETE FROM "agent_evaluations" WHERE "eval_case_id" IS NOT NULL OR "status" = 'Skipped';

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "chk_agent_evaluations_status";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "chk_agent_evaluations_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Failed'));

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_case_score";

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_source";

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "fk_agent_evaluations_eval_case";

--bun:split
ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "fingerprint";

ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "case_score";

ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "judge";

ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "checks";

ALTER TABLE "agent_evaluations" DROP COLUMN IF EXISTS "eval_case_id";

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "subject_id" SET NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "subject_type" SET NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "source_run_id" SET NOT NULL;

--bun:split
DROP TABLE IF EXISTS "agent_eval_cases";
