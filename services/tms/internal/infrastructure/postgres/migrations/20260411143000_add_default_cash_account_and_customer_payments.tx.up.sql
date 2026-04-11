ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS default_cash_account_id VARCHAR(100);

--bun:split
CREATE TABLE IF NOT EXISTS customer_payments(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    customer_id VARCHAR(100) NOT NULL,
    payment_date BIGINT NOT NULL,
    accounting_date BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL,
    applied_amount_minor BIGINT NOT NULL DEFAULT 0,
    unapplied_amount_minor BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    reference_number VARCHAR(100),
    memo TEXT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    posted_batch_id VARCHAR(100),
    reversal_batch_id VARCHAR(100),
    reversed_by_id VARCHAR(100),
    reversed_at BIGINT,
    reversal_reason TEXT,
    created_by_id VARCHAR(100) NOT NULL,
    updated_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    customer_payment_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    applied_amount_minor BIGINT NOT NULL,
    line_number INTEGER NOT NULL,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_customer_payment_applications PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_customer_payment_applications_number UNIQUE (customer_payment_id, organization_id, business_unit_id, line_number),
    CONSTRAINT fk_customer_payment_applications_payment FOREIGN KEY (customer_payment_id, organization_id, business_unit_id) REFERENCES customer_payments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_payment_applications_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id) REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT chk_customer_payment_applications_amount CHECK (applied_amount_minor > 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_customer_payments_customer_date ON customer_payments(organization_id, business_unit_id, customer_id, accounting_date);

--bun:split
CREATE INDEX IF NOT EXISTS idx_customer_payment_applications_invoice ON customer_payment_applications(organization_id, business_unit_id, invoice_id);
