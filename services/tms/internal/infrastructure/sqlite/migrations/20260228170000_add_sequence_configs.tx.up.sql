-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260228170000_add_sequence_configs.tx.up.sql

CREATE TABLE IF NOT EXISTS "sequence_configs"(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "sequence_type" TEXT NOT NULL,
    "prefix" TEXT NOT NULL,
    "include_year" INTEGER NOT NULL DEFAULT 1,
    "year_digits" INTEGER NOT NULL DEFAULT 2 CHECK ("year_digits" BETWEEN 2 AND 4),
    "include_month" INTEGER NOT NULL DEFAULT 1,
    "include_week_number" INTEGER NOT NULL DEFAULT 0,
    "include_day" INTEGER NOT NULL DEFAULT 0,
    "sequence_digits" INTEGER NOT NULL CHECK ("sequence_digits" BETWEEN 1 AND 10),
    "include_location_code" INTEGER NOT NULL DEFAULT 0,
    "location_code" TEXT NOT NULL DEFAULT '',
    "include_random_digits" INTEGER NOT NULL DEFAULT 0,
    "random_digits_count" INTEGER NOT NULL DEFAULT 0 CHECK ("random_digits_count" BETWEEN 0 AND 10),
    "include_check_digit" INTEGER NOT NULL DEFAULT 0,
    "include_business_unit_code" INTEGER NOT NULL DEFAULT 0,
    "business_unit_code" TEXT NOT NULL DEFAULT '',
    "use_separators" INTEGER NOT NULL DEFAULT 0,
    "separator_char" TEXT NOT NULL DEFAULT '',
    "allow_custom_format" INTEGER NOT NULL DEFAULT 0,
    "custom_format" TEXT NOT NULL DEFAULT '',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "fk_sequence_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_sequence_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "uk_sequence_configs_unique_key" UNIQUE ("sequence_type", "organization_id", "business_unit_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sequence_configs_org_type" ON "sequence_configs" ("organization_id", "sequence_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sequence_configs_org_bu_type" ON "sequence_configs" ("organization_id", "business_unit_id", "sequence_type");
