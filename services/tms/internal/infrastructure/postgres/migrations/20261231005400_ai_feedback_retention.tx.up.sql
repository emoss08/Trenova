ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "ai_feedback_retention_period" integer NOT NULL DEFAULT 730;

COMMENT ON COLUMN "data_retention"."ai_feedback_retention_period" IS 'Days a rating of AI output, with its snapshot of what the person saw, is kept before it is deleted. At least 30; 730 by default, and zero reads as the default';

--bun:split

ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_feedback";

--bun:split

ALTER TABLE "data_retention"
    ADD CONSTRAINT "ck_data_retention_ai_feedback" CHECK ("ai_feedback_retention_period" = 0 OR "ai_feedback_retention_period" >= 30);
