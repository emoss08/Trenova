ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

--bun:split
ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel'));
