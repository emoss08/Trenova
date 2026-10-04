DELETE FROM "ai_usage_records"
WHERE "feature" = 'AgentReflection';

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping'));

--bun:split
ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "learning_off";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "learning_off";

--bun:split
DROP INDEX IF EXISTS "idx_agent_memories_supersedes";

--bun:split
DROP INDEX IF EXISTS "idx_agent_memories_reflection";

--bun:split
DELETE FROM "agent_memories"
WHERE "source" = 'Reflection' OR "kind" = 'Procedure';

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_supersedes_self",
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_reflection",
    DROP COLUMN IF EXISTS "supersedes_id",
    DROP COLUMN IF EXISTS "reflection_id";

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision', 'Feedback'));

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_kind";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_kind" CHECK ("kind" IN ('Instruction', 'Fact', 'Correction'));

--bun:split
DROP TABLE IF EXISTS "agent_reflections";
