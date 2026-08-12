-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251104051023_add_fiscal_period.tx.up.sql

CREATE TABLE IF NOT EXISTS "fiscal_periods"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "fiscal_year_id" TEXT NOT NULL,
    "period_number" INTEGER NOT NULL,
    "period_type" TEXT NOT NULL DEFAULT 'Month',
    "status" TEXT NOT NULL DEFAULT 'Inactive',
    "name" TEXT NOT NULL,
    "start_date" INTEGER NOT NULL,
    "end_date" INTEGER NOT NULL,
    "is_adjusting" INTEGER NOT NULL DEFAULT 0,
    "allow_adjusting_entries" INTEGER NOT NULL DEFAULT 0,
    "adjustment_deadline" INTEGER,
    "locked_at" INTEGER,
    "locked_by_id" TEXT,
    "closed_at" INTEGER,
    "closed_by_id" TEXT,
    "reopened_at" INTEGER,
    "reopened_by_id" TEXT,
    "reopen_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fiscal_periods" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_periods_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_periods_locked_by" FOREIGN KEY ("locked_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_periods_reopened_by" FOREIGN KEY ("reopened_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_periods_year_number" UNIQUE ("fiscal_year_id", "period_number"),
    CONSTRAINT "chk_fiscal_periods_dates" CHECK ("end_date" > "start_date")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_fiscal_year ON "fiscal_periods" ("fiscal_year_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_bu_org ON "fiscal_periods" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_dates ON "fiscal_periods" ("start_date", "end_date");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_status ON "fiscal_periods" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_updated_at ON "fiscal_periods" ("updated_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_period_number ON "fiscal_periods" ("period_number");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_is_adjusting ON "fiscal_periods" ("is_adjusting")WHERE
    "is_adjusting" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_open ON "fiscal_periods" ("organization_id", "business_unit_id", "start_date", "end_date")WHERE
    "status" = 'Open';
