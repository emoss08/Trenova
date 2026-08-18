-- Hand-completed SQLite translation of the rate table merge rollback; this
-- file no longer regenerates. See docs/databases.md.
-- Source: 20260927100000_merge_rate_tables_into_matrices.tx.down.sql
--
-- Restores the rate table schema so the code at this revision can run again.
-- Records are NOT restored: nothing distinguishes a migrated single-axis
-- matrix from one somebody built by hand after the merge, so unwinding would
-- move rows a person created. The rates all remain present and readable as
-- matrices; recovering them into these tables requires a database backup
-- taken before the up migration ran.
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
