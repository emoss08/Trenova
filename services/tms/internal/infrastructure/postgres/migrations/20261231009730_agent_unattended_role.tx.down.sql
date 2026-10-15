DROP INDEX IF EXISTS "idx_agent_definitions_unattended_role";

--bun:split
ALTER TABLE "agent_definitions" DROP CONSTRAINT IF EXISTS "fk_agent_definitions_unattended_role";

--bun:split
ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "unattended_role_id";
