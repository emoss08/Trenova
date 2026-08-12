-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260410190000_add_journal_batches.tx.up.sql

CREATE TABLE IF NOT EXISTS "journal_batches"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "batch_number" TEXT NOT NULL,
    "batch_type" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "accounting_date" INTEGER NOT NULL,
    "fiscal_year_id" TEXT NOT NULL,
    "fiscal_period_id" TEXT NOT NULL,
    "entry_count" INTEGER NOT NULL DEFAULT 0,
    "posted_at" INTEGER,
    "posted_by_id" TEXT,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_journal_batches" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_journal_batches_batch_number" UNIQUE ("organization_id", "business_unit_id", "batch_number"),
    CONSTRAINT "fk_journal_batches_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_fiscal_period" FOREIGN KEY ("fiscal_period_id", "organization_id", "business_unit_id") REFERENCES "fiscal_periods"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_posted_by" FOREIGN KEY ("posted_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_batches_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_batches_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split

ALTER TABLE "journal_entries" ADD COLUMN "batch_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_batch_id ON "journal_entries" ("batch_id")WHERE "batch_id" IS NOT NULL;
