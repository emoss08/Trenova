UPDATE "ai_providers"
SET "reasoning_effort" = CASE "reasoning_effort"
    WHEN 'None' THEN 'Off'
    WHEN 'Minimal' THEN 'Low'
    ELSE "reasoning_effort"
END
WHERE "reasoning_effort" IN ('None', 'Minimal');

--bun:split
ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_reasoning_effort";

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_reasoning_effort" CHECK ("reasoning_effort" IN ('Off', 'Low', 'Medium', 'High'));

--bun:split
COMMENT ON COLUMN "ai_providers"."reasoning_effort" IS 'How hard the model is asked to think before answering; Off sends no reasoning parameter';
