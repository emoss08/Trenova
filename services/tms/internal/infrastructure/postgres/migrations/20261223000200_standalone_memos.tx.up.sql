-- Standalone credit and debit memos, and the application of a posted credit
-- memo against the invoices it settles. Applying a credit moves no money: both
-- documents are already on the ledger, so a row only records which open item
-- the credit pays.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "reference_invoice_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "memo_reason" text,
    ADD COLUMN IF NOT EXISTS "memo_kind" varchar(30);

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_reference_invoice" FOREIGN KEY ("reference_invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_memo_kind" CHECK ("memo_kind" IS NULL OR "memo_kind" IN ('Manual', 'LateCharge'));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_reference_invoice" ON "invoices"("reference_invoice_id", "organization_id", "business_unit_id")
WHERE
    "reference_invoice_id" IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS "credit_memo_applications"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "credit_memo_invoice_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "applied_amount_minor" bigint NOT NULL,
    "accounting_date" bigint NOT NULL,
    "line_number" integer NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Applied',
    "unapplied_at" bigint,
    "unapplied_by_id" varchar(100),
    "unapplied_reason" text,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_credit_memo_applications" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_credit_memo_applications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_credit_memo_applications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_credit_memo_applications_credit_memo" FOREIGN KEY ("credit_memo_invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_credit_memo_applications_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_credit_memo_applications_amount" CHECK ("applied_amount_minor" > 0),
    CONSTRAINT "chk_credit_memo_applications_status" CHECK ("status" IN ('Applied', 'Unapplied'))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_credit_memo_applications_invoice" ON "credit_memo_applications"("invoice_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_credit_memo_applications_credit_memo" ON "credit_memo_applications"("credit_memo_invoice_id", "organization_id", "business_unit_id");
