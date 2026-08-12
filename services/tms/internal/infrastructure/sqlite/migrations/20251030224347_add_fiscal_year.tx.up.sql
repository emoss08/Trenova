-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251030224347_add_fiscal_year.tx.up.sql

CREATE TABLE IF NOT EXISTS "fiscal_years"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "year" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "start_date" INTEGER NOT NULL,
    "end_date" INTEGER NOT NULL,
    "is_current" INTEGER NOT NULL DEFAULT 0,
    "is_calendar_year" INTEGER NOT NULL DEFAULT 0,
    "allow_adjusting_entries" INTEGER NOT NULL DEFAULT 0,
    "closed_at" INTEGER,
    "closed_by_id" TEXT,
    "locked_at" INTEGER,
    "locked_by_id" TEXT,
    "reopened_at" INTEGER,
    "reopened_by_id" TEXT,
    "reopen_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fiscal_years" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_years_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_years_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_years_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_years_locked_by" FOREIGN KEY ("locked_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_years_reopened_by" FOREIGN KEY ("reopened_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_years_year" UNIQUE ("organization_id", "year")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_fiscal_years_unique_current ON "fiscal_years" ("organization_id")WHERE
    "is_current" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_bu_org ON "fiscal_years" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_created_updated ON "fiscal_years" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_status ON "fiscal_years" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_year ON "fiscal_years" ("year");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_current ON "fiscal_years" ("is_current")WHERE
    "is_current" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_date_range ON "fiscal_years" ("start_date", "end_date");

--bun:split

CREATE INDEX IF NOT EXISTS idx_fiscal_years_reopened_by ON "fiscal_years" ("reopened_by_id");
