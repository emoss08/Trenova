-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250209211259_pro_number_sequence.tx.up.sql

CREATE TABLE IF NOT EXISTS "pro_number_sequences"(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "year" INTEGER NOT NULL,
    "month" INTEGER NOT NULL,
    "current_sequence" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "version" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "fk_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id"),
    CONSTRAINT "fk_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id"),
    CONSTRAINT "uq_sequence_period" UNIQUE ("organization_id", "business_unit_id", "year", "month")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pro_number_sequences_lookup" ON "pro_number_sequences" ("organization_id", "year", "month");
