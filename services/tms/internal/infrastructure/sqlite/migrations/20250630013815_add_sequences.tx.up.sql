-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250630013815_add_sequences.tx.up.sql

CREATE TABLE IF NOT EXISTS "sequences"(
    "id" TEXT PRIMARY KEY,
    "sequence_type" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "year" INTEGER NOT NULL CHECK (year >= 1900 AND year <= 2100),
    "month" INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    "current_sequence" INTEGER NOT NULL DEFAULT 0 CHECK (current_sequence >= 0),
    "last_generated" TEXT NOT NULL DEFAULT '',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "fk_sequences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_sequences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "uk_sequences_unique_key" UNIQUE ("sequence_type", "organization_id", "business_unit_id", "year", "month")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sequences_org_type" ON sequences ("organization_id", "sequence_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sequences_org_bu_type" ON sequences ("organization_id", "business_unit_id", "sequence_type")WHERE
    "business_unit_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sequences_year_month" ON sequences ("year", "month");
