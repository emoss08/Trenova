-- A dispute is a case, not a flag: who raised it, why, for how much, and how
-- it ended. The invoice's dispute_status stays as the quick read and is kept
-- in step with the open case, so an invoice is Disputed while a case is Open.
CREATE TABLE IF NOT EXISTS "invoice_disputes"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Open',
    "reason_code" varchar(40) NOT NULL,
    "disputed_amount" numeric(19, 4) NOT NULL,
    "disputed_amount_minor" bigint NOT NULL,
    "notes" text,
    "opened_by_id" varchar(100) NOT NULL,
    "opened_at" bigint NOT NULL,
    "resolved_by_id" varchar(100),
    "resolved_at" bigint,
    "resolution" varchar(30),
    "resolution_adjustment_id" varchar(100),
    "resolution_notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_invoice_disputes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_invoice_disputes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_disputes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_disputes_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_disputes_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_invoice_disputes_adjustment" FOREIGN KEY ("resolution_adjustment_id", "organization_id", "business_unit_id") REFERENCES "invoice_adjustments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_invoice_disputes_status" CHECK ("status" IN ('Open', 'Resolved', 'Withdrawn')),
    CONSTRAINT "chk_invoice_disputes_reason" CHECK ("reason_code" IN ('RateDiscrepancy', 'AccessorialDisputed', 'ServiceFailure', 'DuplicateBilling', 'WrongBillTo', 'MissingDocumentation', 'Other')),
    CONSTRAINT "chk_invoice_disputes_resolution" CHECK ("resolution" IS NULL OR "resolution" IN ('CreditIssued', 'InvoiceUpheld', 'Rebilled', 'WrittenOff', 'CustomerWithdrew')),
    CONSTRAINT "chk_invoice_disputes_amount" CHECK ("disputed_amount" > 0 AND "disputed_amount_minor" > 0),
    CONSTRAINT "chk_invoice_disputes_resolved" CHECK (("status" = 'Resolved' AND "resolution" IS NOT NULL AND "resolved_at" IS NOT NULL) OR "status" <> 'Resolved')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_disputes_open" ON "invoice_disputes"("invoice_id", "organization_id", "business_unit_id")
WHERE
    "status" = 'Open';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_disputes_invoice" ON "invoice_disputes"("invoice_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_disputes_customer_status" ON "invoice_disputes"("customer_id", "status", "organization_id", "business_unit_id");
