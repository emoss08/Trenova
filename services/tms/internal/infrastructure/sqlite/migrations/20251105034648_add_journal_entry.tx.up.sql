-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251105034648_add_journal_entry.tx.up.sql

CREATE TABLE IF NOT EXISTS "journal_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "fiscal_year_id" TEXT NOT NULL,
    "fiscal_period_id" TEXT NOT NULL,
    "entry_number" TEXT NOT NULL,
    "entry_date" INTEGER NOT NULL,
    "entry_type" TEXT NOT NULL DEFAULT 'Standard',
    "accounting_date" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "reference_number" TEXT,
    "reference_type" TEXT,
    "reference_id" TEXT,
    "description" TEXT NOT NULL,
    "total_debit" INTEGER NOT NULL DEFAULT 0,
    "total_credit" INTEGER NOT NULL DEFAULT 0,
    "is_posted" INTEGER NOT NULL DEFAULT 0,
    "posted_at" INTEGER,
    "posted_by_id" TEXT,
    "is_auto_generated" INTEGER NOT NULL DEFAULT 0,
    "is_reversal" INTEGER NOT NULL DEFAULT 0,
    "reversal_of_id" TEXT,
    "reversed_by_id" TEXT,
    "reversal_date" INTEGER,
    "reversal_reason" TEXT,
    "requires_approval" INTEGER NOT NULL DEFAULT 1,
    "is_approved" INTEGER NOT NULL DEFAULT 0,
    "approved_at" INTEGER,
    "approved_by_id" TEXT,
    "approval_notes" TEXT,
    "is_rejected" INTEGER NOT NULL DEFAULT 0,
    "rejected_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejection_reason" TEXT,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_journal_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_journal_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_entries_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entries_fiscal_period" FOREIGN KEY ("fiscal_period_id", "organization_id", "business_unit_id") REFERENCES "fiscal_periods"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entries_posted_by" FOREIGN KEY ("posted_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_entries_approved_by" FOREIGN KEY ("approved_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_entries_reversal_of" FOREIGN KEY ("reversal_of_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entries_reversed_by" FOREIGN KEY ("reversed_by_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entries_rejected_by" FOREIGN KEY ("rejected_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_entries_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entries_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_journal_entries_entry_number" UNIQUE ("organization_id", "business_unit_id", "entry_number"),
    CONSTRAINT "chk_journal_entries_balanced" CHECK ("total_debit" = "total_credit" OR "status" = 'Draft'),
    CONSTRAINT "chk_journal_entries_posted_at" CHECK (("is_posted" = TRUE AND "posted_at" IS NOT NULL AND "posted_by_id" IS NOT NULL) OR ("is_posted" = FALSE)),
    CONSTRAINT "chk_journal_entries_approved_at" CHECK (("is_approved" = TRUE AND "approved_at" IS NOT NULL AND "approved_by_id" IS NOT NULL) OR ("is_approved" = FALSE)),
    CONSTRAINT "chk_journal_entries_reversal" CHECK (("is_reversal" = TRUE AND "reversal_of_id" IS NOT NULL) OR ("is_reversal" = FALSE)),
    CONSTRAINT "chk_journal_entries_no_self_reversal" CHECK ("id" != "reversal_of_id" AND "id" != "reversed_by_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_bu_org ON "journal_entries" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_created_updated ON "journal_entries" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_status ON "journal_entries" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_type ON "journal_entries" ("entry_type");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_entry_date ON "journal_entries" ("entry_date");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_fiscal_year ON "journal_entries" ("fiscal_year_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_fiscal_period ON "journal_entries" ("fiscal_period_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_posted ON "journal_entries" ("is_posted")WHERE
    "is_posted" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_approved ON "journal_entries" ("is_approved")WHERE
    "is_approved" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_reversal ON "journal_entries" ("is_reversal")WHERE
    "is_reversal" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_reference ON "journal_entries" ("reference_type", "reference_id")WHERE
    "reference_type" IS NOT NULL AND "reference_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entries_entry_number ON "journal_entries" ("entry_number");

--bun:split

CREATE TABLE IF NOT EXISTS "journal_entry_lines"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "journal_entry_id" TEXT NOT NULL,
    "gl_account_id" TEXT NOT NULL,
    "line_number" INTEGER NOT NULL,
    "description" TEXT NOT NULL,
    "debit_amount" INTEGER NOT NULL DEFAULT 0,
    "credit_amount" INTEGER NOT NULL DEFAULT 0,
    "net_amount" INTEGER NOT NULL DEFAULT 0,
    "department_id" TEXT,
    "project_id" TEXT,
    "location_id" TEXT,
    "customer_id" TEXT,
    "vendor_id" TEXT,
    "tax_code" TEXT,
    "tax_amount" INTEGER NOT NULL DEFAULT 0,
    "is_tax_line" INTEGER NOT NULL DEFAULT 0,
    "is_reconciled" INTEGER NOT NULL DEFAULT 0,
    "reconciled_at" INTEGER,
    "reconciled_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_journal_entry_lines" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_journal_entry_lines_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_entry_lines_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_entry_lines_journal_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_journal_entry_lines_gl_account" FOREIGN KEY ("gl_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_journal_entry_lines_location" FOREIGN KEY ("location_id", "organization_id", "business_unit_id") REFERENCES "locations"("id", "organization_id", "business_unit_id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_entry_lines_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE SET NULL,
    CONSTRAINT "fk_journal_entry_lines_reconciled_by" FOREIGN KEY ("reconciled_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_journal_entry_lines_line_number" UNIQUE ("journal_entry_id", "organization_id", "business_unit_id", "line_number"),
    CONSTRAINT "chk_journal_entry_lines_debit_or_credit" CHECK (("debit_amount" > 0 AND "credit_amount" = 0) OR ("credit_amount" > 0 AND "debit_amount" = 0)),
    CONSTRAINT "chk_journal_entry_lines_reconciled_at" CHECK (("is_reconciled" = TRUE AND "reconciled_at" IS NOT NULL AND "reconciled_by_id" IS NOT NULL) OR ("is_reconciled" = FALSE))
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_bu_org ON "journal_entry_lines" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_created_updated ON "journal_entry_lines" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_journal_entry ON "journal_entry_lines" ("journal_entry_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_gl_account ON "journal_entry_lines" ("gl_account_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_location ON "journal_entry_lines" ("location_id")WHERE
    "location_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_customer ON "journal_entry_lines" ("customer_id")WHERE
    "customer_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_journal_entry_lines_reconciled ON "journal_entry_lines" ("is_reconciled")WHERE
    "is_reconciled" = TRUE;
