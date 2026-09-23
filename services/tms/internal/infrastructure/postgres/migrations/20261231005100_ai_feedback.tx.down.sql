DROP INDEX IF EXISTS "idx_ai_feedback_thread";

--bun:split
DROP INDEX IF EXISTS "idx_ai_feedback_created";

--bun:split
DROP INDEX IF EXISTS "idx_ai_feedback_negative_pattern";

--bun:split
DROP INDEX IF EXISTS "idx_ai_feedback_agent_created";

--bun:split
DROP TABLE IF EXISTS "ai_feedback";
