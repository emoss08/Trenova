-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005800_agent_extensions.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_extensions" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "availability" TEXT NOT NULL DEFAULT 'SelectedAgents',
    "configuration" TEXT NOT NULL DEFAULT '{}',
    "enabled_by_id" TEXT,
    "enabled_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_extensions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_extensions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extensions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extensions_enabled_by" FOREIGN KEY ("enabled_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "uq_agent_extensions_tenant_type" UNIQUE ("organization_id", "business_unit_id", "type"),
    CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa'))
);

--bun:split

CREATE TABLE IF NOT EXISTS "agent_extension_usage_daily" (
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "extension_type" TEXT NOT NULL,
    "day" INTEGER NOT NULL,
    "requests" INTEGER NOT NULL DEFAULT 0,
    "failures" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" REAL NOT NULL DEFAULT 0,
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_extension_usage_daily" PRIMARY KEY ("organization_id", "business_unit_id", "extension_type", "day"),
    CONSTRAINT "fk_agent_extension_usage_daily_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_extension_usage_daily_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_extension_usage_daily_requests" CHECK ("requests" >= 0 AND "failures" >= 0 AND "cost_usd" >= 0)
);
