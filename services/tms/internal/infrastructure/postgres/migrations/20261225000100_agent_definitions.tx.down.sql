DROP INDEX IF EXISTS "idx_agent_definitions_due";

DROP INDEX IF EXISTS "idx_agent_definitions_lookup";

DROP INDEX IF EXISTS "uq_agent_definitions_system_key";

DROP INDEX IF EXISTS "uq_agent_definitions_org_name";

--bun:split
DROP TABLE IF EXISTS "agent_definitions";
