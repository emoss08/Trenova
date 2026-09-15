-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: the enum becomes a
-- TEXT column. Hand-edits are preserved only if you stop regenerating this
-- file; see docs/databases.md.
-- Source: 20261223000400_late_charges.tx.up.sql

ALTER TABLE "billing_controls" ADD COLUMN "late_charge_assessment_mode" TEXT NOT NULL DEFAULT 'Disabled' CHECK ("late_charge_assessment_mode" IN ('Disabled', 'Preview', 'Automatic'));

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "late_charge_minimum_amount" REAL NOT NULL DEFAULT 0;

--bun:split

CREATE TABLE IF NOT EXISTS "late_charge_assessments"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "source_invoice_id" TEXT NOT NULL,
    "period_index" INTEGER NOT NULL,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "as_of_date" INTEGER NOT NULL,
    "basis_open_balance_minor" INTEGER NOT NULL,
    "rate_percent" REAL NOT NULL,
    "charge_minor" INTEGER NOT NULL,
    "debit_memo_invoice_id" TEXT,
    "debit_memo_line_id" TEXT,
    "run_key" TEXT NOT NULL,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_late_charge_assessments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_late_charge_assessments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_late_charge_assessments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_late_charge_assessments_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_late_charge_assessments_source_invoice" FOREIGN KEY ("source_invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_late_charge_assessments_debit_memo" FOREIGN KEY ("debit_memo_invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_late_charge_assessments_period" CHECK ("period_index" >= 1 AND "period_end" >= "period_start"),
    CONSTRAINT "chk_late_charge_assessments_charge" CHECK ("charge_minor" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_late_charge_assessments_invoice_period" ON "late_charge_assessments" ("organization_id", "business_unit_id", "source_invoice_id", "period_index");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_customer" ON "late_charge_assessments" ("customer_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_debit_memo" ON "late_charge_assessments" ("debit_memo_invoice_id", "organization_id", "business_unit_id")WHERE
    "debit_memo_invoice_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_run_key" ON "late_charge_assessments" ("run_key");
