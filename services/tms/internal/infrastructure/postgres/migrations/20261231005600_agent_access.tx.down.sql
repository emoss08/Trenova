DROP TABLE IF EXISTS "role_agent_grants";

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_system_access";

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_access_mode";

--bun:split

ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "access_mode";
