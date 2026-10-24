-- An agent that runs on its own, on a schedule or an event, acts under the
-- fixed ceiling every agent shares. An organization may also name a role it
-- runs as: the agent may then do only what both the ceiling and that role
-- allow, and the role's editor and audit trail decide its reach.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "unattended_role_id" varchar(100);

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "fk_agent_definitions_unattended_role" FOREIGN KEY ("unattended_role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_unattended_role" ON "agent_definitions"("unattended_role_id")
WHERE "unattended_role_id" IS NOT NULL;

COMMENT ON COLUMN agent_definitions.unattended_role_id IS 'The role an agent runs as when no person is present. Its unattended checks pass only where both the agent ceiling and this role allow; null means the ceiling alone.';
