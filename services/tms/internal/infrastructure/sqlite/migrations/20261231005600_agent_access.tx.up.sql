-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005600_agent_access.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "access_mode" TEXT NOT NULL DEFAULT 'Everyone';

--bun:split

CREATE TABLE IF NOT EXISTS "role_agent_grants" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "role_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "granted_by" TEXT,
    "granted_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
