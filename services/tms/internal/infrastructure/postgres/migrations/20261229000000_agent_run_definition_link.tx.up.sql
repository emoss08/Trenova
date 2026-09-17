-- A run now records which definition produced it and how it started, so the
-- agents screen can show each agent's history and the sweep can count what is
-- still open per definition.
ALTER TABLE "agent_runs"
    ADD COLUMN IF NOT EXISTS "agent_definition_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "trigger" varchar(20) NOT NULL DEFAULT 'Manual',
    ADD COLUMN IF NOT EXISTS "summary" text;

--bun:split
ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_trigger" CHECK ("trigger" IN ('Manual', 'Chat', 'Scheduled', 'Event', 'Continuous'));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_runs_definition" ON "agent_runs"("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC);
