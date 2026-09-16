-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261225000100_agent_definitions.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_definitions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL,
    "focus" TEXT,
    "tool_names" TEXT,
    "autonomy_ceiling" TEXT NOT NULL DEFAULT 'Propose',
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_definitions_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definitions_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_definitions_kind" CHECK ("kind" IN ('DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant', 'GeneralAssistant')),
    CONSTRAINT "ck_agent_definitions_autonomy" CHECK ("autonomy_ceiling" IN ('Propose', 'ActWithApproval', 'AutoExecute')),
    CONSTRAINT "ck_agent_definitions_focus_length" CHECK ("focus" IS NULL OR length("focus") <= 2000)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_org_name" ON "agent_definitions" ("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_lookup" ON "agent_definitions" ("organization_id", "business_unit_id", "enabled");
