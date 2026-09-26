ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_correction";

--bun:split
ALTER TABLE "data_retention" DROP COLUMN IF EXISTS "ai_correction_retention_period";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "ai_training_consent_changed_by_id",
    DROP COLUMN IF EXISTS "ai_training_consent_changed_at",
    DROP COLUMN IF EXISTS "ai_training_consent";

--bun:split
DROP INDEX IF EXISTS "idx_ai_corrections_subject";

--bun:split
DROP INDEX IF EXISTS "idx_ai_corrections_task_kind";

--bun:split
DROP INDEX IF EXISTS "idx_ai_corrections_created";

--bun:split
DROP TABLE IF EXISTS "ai_corrections";
