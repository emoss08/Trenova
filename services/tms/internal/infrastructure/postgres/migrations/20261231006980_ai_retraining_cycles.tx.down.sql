DROP INDEX IF EXISTS "uq_ai_retraining_cycles_open";

--bun:split
DROP INDEX IF EXISTS "idx_ai_retraining_cycles_created";

--bun:split
DROP TABLE IF EXISTS "ai_retraining_cycles";
