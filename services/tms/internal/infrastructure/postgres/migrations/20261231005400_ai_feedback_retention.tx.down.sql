ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_feedback";

--bun:split

ALTER TABLE "data_retention" DROP COLUMN IF EXISTS "ai_feedback_retention_period";
