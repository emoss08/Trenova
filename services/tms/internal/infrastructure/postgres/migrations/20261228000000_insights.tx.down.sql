DROP INDEX IF EXISTS "idx_insights_detector";

--bun:split
DROP INDEX IF EXISTS "idx_insights_dismissed_dedupe";

--bun:split
DROP INDEX IF EXISTS "idx_insights_active";

--bun:split
DROP INDEX IF EXISTS "uq_insights_active_dedupe";

--bun:split
DROP TABLE IF EXISTS "insights";
