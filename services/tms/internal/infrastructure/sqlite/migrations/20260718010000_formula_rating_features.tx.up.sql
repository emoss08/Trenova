-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260718010000_formula_rating_features.tx.up.sql

ALTER TABLE "formula_templates" ADD COLUMN "breakdown_definitions" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "min_charge" REAL;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "max_charge" REAL;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "submitted_by_id" TEXT;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "submitted_at" INTEGER;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "approved_by_id" TEXT;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "approved_at" INTEGER;

--bun:split

ALTER TABLE "formula_templates" ADD COLUMN "review_comment" TEXT;

--bun:split

ALTER TABLE "formula_template_versions" ADD COLUMN "breakdown_definitions" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "formula_template_versions" ADD COLUMN "min_charge" REAL;

--bun:split

ALTER TABLE "formula_template_versions" ADD COLUMN "max_charge" REAL;

--bun:split

ALTER TABLE "formula_template_versions" ADD COLUMN "effective_from" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_effective"
    ON "formula_template_versions" ("template_id", "organization_id", "business_unit_id", "effective_from" DESC)WHERE "effective_from" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "rate_tables"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "description" TEXT,
    "lookup_type" TEXT NOT NULL,
    "active" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_tables" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_tables_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_tables_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_tables_key" ON "rate_tables" ("organization_id", "business_unit_id", "key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_tables_bu_org" ON "rate_tables" ("business_unit_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_table_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_table_id" TEXT NOT NULL,
    "match_key" TEXT,
    "range_min" REAL,
    "range_max" REAL,
    "value" REAL NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_table_entries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_table_entries_rate_table" FOREIGN KEY ("rate_table_id", "business_unit_id", "organization_id") REFERENCES "rate_tables"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_table_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_table_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_table_entries_table" ON "rate_table_entries" ("rate_table_id", "organization_id", "business_unit_id");
