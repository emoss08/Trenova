DELETE FROM "agent_extension_usage_daily" WHERE "extension_type" = 'Parallel';

--bun:split
DELETE FROM "agent_extensions" WHERE "type" = 'Parallel';

--bun:split
ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

--bun:split
ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa'));
