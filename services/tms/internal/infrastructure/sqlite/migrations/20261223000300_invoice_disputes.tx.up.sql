-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed. Hand-edits are
-- preserved only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261223000300_invoice_disputes.tx.up.sql

CREATE TABLE IF NOT EXISTS "invoice_disputes"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "reason_code" TEXT NOT NULL,
    "disputed_amount" REAL NOT NULL,
    "disputed_amount_minor" INTEGER NOT NULL,
    "notes" TEXT,
    "opened_by_id" TEXT NOT NULL,
    "opened_at" INTEGER NOT NULL,
    "resolved_by_id" TEXT,
    "resolved_at" INTEGER,
    "resolution" TEXT,
    "resolution_adjustment_id" TEXT,
    "resolution_notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_disputes_open" ON "invoice_disputes" ("invoice_id", "organization_id", "business_unit_id")WHERE
    "status" = 'Open';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_disputes_invoice" ON "invoice_disputes" ("invoice_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_disputes_customer_status" ON "invoice_disputes" ("customer_id", "status", "organization_id", "business_unit_id");
