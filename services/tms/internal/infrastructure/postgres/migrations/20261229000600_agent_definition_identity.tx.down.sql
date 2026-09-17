ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "chk_agent_definitions_accent";

--bun:split
ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "icon",
    DROP COLUMN IF EXISTS "accent";
