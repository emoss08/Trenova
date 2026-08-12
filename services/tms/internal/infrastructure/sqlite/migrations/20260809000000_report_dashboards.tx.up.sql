-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260809000000_report_dashboards.tx.up.sql

CREATE TABLE IF NOT EXISTS "report_dashboards"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT,
    "tags" TEXT,
    "owner_id" TEXT NOT NULL,
    "visibility" TEXT NOT NULL DEFAULT 'private',
    "layout" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_report_dashboards" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_dashboards_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_dashboards_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_dashboards_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_dashboards_visibility" CHECK ("visibility" IN ('private', 'shared'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_dashboards_tenant" ON "report_dashboards" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_dashboards_owner" ON "report_dashboards" ("owner_id", "organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_report_dashboards_name" ON "report_dashboards" (lower("name"), "owner_id", "organization_id", "business_unit_id");
