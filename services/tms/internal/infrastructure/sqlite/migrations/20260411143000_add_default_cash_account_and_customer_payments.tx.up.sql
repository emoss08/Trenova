-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411143000_add_default_cash_account_and_customer_payments.tx.up.sql

ALTER TABLE "accounting_controls" ADD COLUMN "default_cash_account_id" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS customer_payments(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "payment_date" INTEGER NOT NULL,
    "accounting_date" INTEGER NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "applied_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "unapplied_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL,
    "payment_method" TEXT NOT NULL,
    "reference_number" TEXT,
    "memo" TEXT,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "posted_batch_id" TEXT,
    "reversal_batch_id" TEXT,
    "reversed_by_id" TEXT,
    "reversed_at" INTEGER,
    "reversal_reason" TEXT,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_customer_payments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_customer_payments_customer FOREIGN KEY (customer_id, organization_id, business_unit_id) REFERENCES customers(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_customer_payments_batch FOREIGN KEY (posted_batch_id, organization_id, business_unit_id) REFERENCES journal_batches(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_customer_payments_reversal_batch FOREIGN KEY (reversal_batch_id, organization_id, business_unit_id) REFERENCES journal_batches(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_customer_payments_reversed_by FOREIGN KEY (reversed_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_customer_payments_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_customer_payments_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_customer_payments_amounts CHECK (amount_minor > 0 AND applied_amount_minor >= 0 AND unapplied_amount_minor >= 0)
);

--bun:split

CREATE TABLE IF NOT EXISTS customer_payment_applications(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "customer_payment_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "applied_amount_minor" INTEGER NOT NULL,
    "short_pay_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "line_number" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_customer_payment_applications PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_customer_payment_applications_number UNIQUE (customer_payment_id, organization_id, business_unit_id, line_number),
    CONSTRAINT fk_customer_payment_applications_payment FOREIGN KEY (customer_payment_id, organization_id, business_unit_id) REFERENCES customer_payments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_payment_applications_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id) REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT chk_customer_payment_applications_amount CHECK (applied_amount_minor > 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_customer_payments_customer_date ON customer_payments (organization_id, business_unit_id, customer_id, accounting_date);

--bun:split

CREATE INDEX IF NOT EXISTS idx_customer_payment_applications_invoice ON customer_payment_applications (organization_id, business_unit_id, invoice_id);
