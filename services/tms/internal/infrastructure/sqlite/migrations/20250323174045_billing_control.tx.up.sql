-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250323174045_billing_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "billing_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "invoice_number_prefix" TEXT NOT NULL DEFAULT 'INV-',
    "credit_memo_number_prefix" TEXT NOT NULL DEFAULT 'CM-',
    "payment_term" TEXT NOT NULL DEFAULT 'Net30',
    "show_invoice_due_date" INTEGER NOT NULL DEFAULT 1,
    "invoice_terms" TEXT,
    "invoice_footer" TEXT,
    "show_amount_due" INTEGER NOT NULL DEFAULT 1,
    "auto_transfer" INTEGER NOT NULL DEFAULT 1,
    "transfer_schedule" TEXT NOT NULL DEFAULT 'Continuous',
    "transfer_batch_size" INTEGER NOT NULL DEFAULT 100 CHECK ("transfer_batch_size" >= 1),
    "auto_mark_ready_to_bill" INTEGER NOT NULL DEFAULT 1,
    "enforce_customer_billing_req" INTEGER NOT NULL DEFAULT 1,
    "validate_customer_rates" INTEGER NOT NULL DEFAULT 0,
    "auto_bill" INTEGER NOT NULL DEFAULT 0,
    "send_auto_bill_notifications" INTEGER NOT NULL DEFAULT 1,
    "auto_bill_batch_size" INTEGER NOT NULL DEFAULT 100 CHECK ("auto_bill_batch_size" >= 1),
    "billing_exception_handling" TEXT NOT NULL DEFAULT 'Queue',
    "rate_discrepancy_threshold" REAL NOT NULL DEFAULT 5.00 CHECK ("rate_discrepancy_threshold" >= 0),
    "auto_resolve_minor_discrepancies" INTEGER NOT NULL DEFAULT 0,
    "allow_invoice_consolidation" INTEGER NOT NULL DEFAULT 0,
    "consolidation_period_days" INTEGER NOT NULL DEFAULT 7 CHECK ("consolidation_period_days" >= 1),
    "group_consolidated_invoices" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_billing_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_billing_controls_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_controls_business_unit" ON "billing_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_controls_created_at" ON "billing_controls" ("created_at", "updated_at");
