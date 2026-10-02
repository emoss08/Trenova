-- A model that reasons by default keeps reasoning at its own default when no
-- reasoning parameter is sent, so Off cannot stop it thinking. None tells it
-- not to reason and Minimal asks for the least it allows.
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_reasoning_effort";

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_reasoning_effort" CHECK ("reasoning_effort" IN ('Off', 'None', 'Minimal', 'Low', 'Medium', 'High'));

--bun:split
COMMENT ON COLUMN "ai_providers"."reasoning_effort" IS 'How hard the model is asked to think before answering; Off sends no reasoning parameter, None tells the model not to reason';
