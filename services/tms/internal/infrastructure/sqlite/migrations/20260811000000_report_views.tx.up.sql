-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260811000000_report_views.tx.up.sql

CREATE TABLE IF NOT EXISTS "report_views"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "definition_id" TEXT NOT NULL,
    "owner_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "params" TEXT,
    "shared" INTEGER NOT NULL DEFAULT 0,
    "pinned" INTEGER NOT NULL DEFAULT 0,
    "format" TEXT,
    "last_run_at" INTEGER,
    "run_count" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_report_views" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_views_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_views_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_views_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_views_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_views_tenant" ON "report_views" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_views_definition" ON "report_views" ("definition_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_views_owner" ON "report_views" ("owner_id", "organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_report_views_name" ON "report_views" (lower("name"), "definition_id", "owner_id", "organization_id", "business_unit_id");
