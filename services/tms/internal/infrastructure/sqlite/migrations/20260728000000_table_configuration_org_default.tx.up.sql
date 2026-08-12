-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260728000000_table_configuration_org_default.tx.up.sql

ALTER TABLE "table_configurations" ADD COLUMN "is_org_default" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_table_configurations_org_default"
    ON "table_configurations" ("organization_id", "business_unit_id", "resource")WHERE "is_org_default" = TRUE;
