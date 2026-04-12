CREATE TABLE IF NOT EXISTS bank_receipts(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    receipt_date BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL,
    reference_number VARCHAR(100),
    memo TEXT,
    status VARCHAR(50) NOT NULL,
    matched_customer_payment_id VARCHAR(100),
    matched_at BIGINT,
    matched_by_id VARCHAR(100),
    exception_reason TEXT,
    created_by_id VARCHAR(100) NOT NULL,
    updated_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_bank_receipts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipts_payment FOREIGN KEY (matched_customer_payment_id, organization_id, business_unit_id) REFERENCES customer_payments(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipts_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipts_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipts_matched_by FOREIGN KEY (matched_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_bank_receipts_amount CHECK (amount_minor > 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_bank_receipts_status_date ON bank_receipts(organization_id, business_unit_id, status, receipt_date);
