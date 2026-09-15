-- Late charges are assessed by a nightly run rather than typed by hand. The
-- organization chooses whether the run only previews, or raises and posts the
-- debit memos itself; the rate and grace period stay on each customer's
-- billing profile. Every (invoice, period) is assessed at most once, which the
-- unique index below guarantees even across concurrent runs.
DO $$
BEGIN
    CREATE TYPE "late_charge_assessment_mode_enum" AS ENUM (
        'Disabled',
        'Preview',
        'Automatic'
    );
EXCEPTION
    WHEN duplicate_object THEN
        NULL;
END
$$;

--bun:split
ALTER TABLE "billing_controls"
    ADD COLUMN IF NOT EXISTS "late_charge_assessment_mode" late_charge_assessment_mode_enum NOT NULL DEFAULT 'Disabled',
    ADD COLUMN IF NOT EXISTS "late_charge_minimum_amount" numeric(19, 4) NOT NULL DEFAULT 0;

--bun:split
CREATE TABLE IF NOT EXISTS "late_charge_assessments"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "source_invoice_id" varchar(100) NOT NULL,
    "period_index" integer NOT NULL,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "as_of_date" bigint NOT NULL,
    "basis_open_balance_minor" bigint NOT NULL,
    "rate_percent" numeric(9, 4) NOT NULL,
    "charge_minor" bigint NOT NULL,
    "debit_memo_invoice_id" varchar(100),
    "debit_memo_line_id" varchar(100),
    "run_key" varchar(100) NOT NULL,
    "created_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_late_charge_assessments_invoice_period" ON "late_charge_assessments"("organization_id", "business_unit_id", "source_invoice_id", "period_index");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_customer" ON "late_charge_assessments"("customer_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_debit_memo" ON "late_charge_assessments"("debit_memo_invoice_id", "organization_id", "business_unit_id")
WHERE
    "debit_memo_invoice_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_late_charge_assessments_run_key" ON "late_charge_assessments"("run_key");
