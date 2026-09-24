ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_data_access_ceiling";

--bun:split

ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "data_access_ceiling";
