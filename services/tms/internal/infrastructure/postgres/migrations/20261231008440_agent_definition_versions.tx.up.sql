-- An agent as it stood after each save, so the builder can show its history
-- and load an earlier version as a draft, and a save that lost a race can say
-- what the other save changed.
CREATE TABLE IF NOT EXISTS "agent_definition_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "version" bigint NOT NULL,
    "snapshot" jsonb NOT NULL,
    "author_id" varchar(100),
    "summary" varchar(500),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_definition_versions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_definition_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definition_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definition_versions_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definition_versions_author" FOREIGN KEY ("author_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
-- One row per saved version of an agent, newest first when the builder lists
-- its history.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definition_versions_version" ON "agent_definition_versions"("organization_id", "business_unit_id", "agent_definition_id", "version" DESC);

--bun:split
COMMENT ON TABLE "agent_definition_versions" IS 'An agent as it stood after each save, for its history, restoring a draft and explaining a save conflict';

--bun:split
SELECT trenova_rls.reconcile();
