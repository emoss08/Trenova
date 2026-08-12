-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250829213302_add_data_retention.tx.up.sql

CREATE TABLE IF NOT EXISTS "data_retention"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "audit_retention_period" INTEGER NOT NULL DEFAULT 120,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_data_retention" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dr_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dr_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "uq_data_retention_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_data_retention_bu_org" ON "data_retention" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_data_rention_timestamps" ON "data_retention" ("created_at", "updated_at");
