-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015359_business_unit.tx.up.sql

CREATE TABLE IF NOT EXISTS "business_units"(
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_business_units_code" ON "business_units" (lower("code"));
