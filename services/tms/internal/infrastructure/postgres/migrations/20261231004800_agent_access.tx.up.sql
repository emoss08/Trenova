ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "access_mode" VARCHAR(20) NOT NULL DEFAULT 'Everyone';

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_access_mode";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_access_mode" CHECK ("access_mode" IN ('Everyone', 'Roles'));

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_system_access";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_system_access" CHECK ("system_key" IS NULL OR "access_mode" = 'Everyone');

COMMENT ON COLUMN "agent_definitions"."access_mode" IS 'Who may use the agent among the people who may use the assistant: Everyone, or only the roles granted it in role_agent_grants. A system agent is always Everyone, and a restricted agent with no grants is usable by nobody';

--bun:split

CREATE TABLE IF NOT EXISTS "role_agent_grants" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "granted_by" varchar(100),
    "granted_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_role_agent_grants" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_agent_grants_role" FOREIGN KEY ("role_id", "business_unit_id", "organization_id") REFERENCES "roles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_agent_grants_agent_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_agent_grants_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_agent_grants_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_agent_grants_granted_by" FOREIGN KEY ("granted_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_role_agent_grants_role_agent"
    ON "role_agent_grants" ("role_id", "agent_definition_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_role_agent_grants_role"
    ON "role_agent_grants" ("organization_id", "role_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_role_agent_grants_agent"
    ON "role_agent_grants" ("organization_id", "business_unit_id", "agent_definition_id");

COMMENT ON TABLE "role_agent_grants" IS 'Roles allowed to use an agent whose access_mode is Roles; a role grants its agents to every role that inherits it';

COMMENT ON COLUMN "role_agent_grants"."role_id" IS 'The role whose holders, directly or through a role that inherits it, may use the agent';

COMMENT ON COLUMN "role_agent_grants"."agent_definition_id" IS 'The agent the role may use; the grant is not consulted while the agent is open to everyone';

COMMENT ON COLUMN "role_agent_grants"."granted_by" IS 'The person who granted it, cleared if their account is deleted';

COMMENT ON COLUMN "role_agent_grants"."granted_at" IS 'When the role was granted the agent';
