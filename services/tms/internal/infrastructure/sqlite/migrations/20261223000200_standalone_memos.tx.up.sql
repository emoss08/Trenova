-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite has no
-- ALTER TABLE ADD CONSTRAINT, so the reference-invoice foreign key and the
-- memo-kind check are enforced by the application here. Hand-edits are
-- preserved only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261223000200_standalone_memos.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "reference_invoice_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "memo_reason" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "memo_kind" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_reference_invoice" ON "invoices" ("reference_invoice_id", "organization_id", "business_unit_id")WHERE
    "reference_invoice_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "credit_memo_applications"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "credit_memo_invoice_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "applied_amount_minor" INTEGER NOT NULL,
    "accounting_date" INTEGER NOT NULL,
    "line_number" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Applied',
    "unapplied_at" INTEGER,
    "unapplied_by_id" TEXT,
    "unapplied_reason" TEXT,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_credit_memo_applications" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_credit_memo_applications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_credit_memo_applications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_credit_memo_applications_credit_memo" FOREIGN KEY ("credit_memo_invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_credit_memo_applications_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_credit_memo_applications_amount" CHECK ("applied_amount_minor" > 0),
    CONSTRAINT "chk_credit_memo_applications_status" CHECK ("status" IN ('Applied', 'Unapplied'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_credit_memo_applications_invoice" ON "credit_memo_applications" ("invoice_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_credit_memo_applications_credit_memo" ON "credit_memo_applications" ("credit_memo_invoice_id", "organization_id", "business_unit_id");
