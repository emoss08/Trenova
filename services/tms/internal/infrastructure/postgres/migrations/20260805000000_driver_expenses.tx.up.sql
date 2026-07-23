CREATE TABLE IF NOT EXISTS driver_expenses(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    shipment_id VARCHAR(100),
    pay_code_id VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'Pending',
    amount_minor BIGINT NOT NULL,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    description VARCHAR(255) NOT NULL,
    incurred_date BIGINT NOT NULL,
    receipt_document_id VARCHAR(100),
    submitted_by_user_id VARCHAR(100) NOT NULL,
    review_note TEXT,
    reviewed_by_id VARCHAR(100),
    reviewed_at BIGINT,
    settlement_line_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_expenses PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_expenses_status CHECK (status IN ('Pending', 'Approved', 'Rejected', 'Reimbursed', 'Cancelled')),
    CONSTRAINT chk_driver_expenses_amount CHECK (amount_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (pay_code_id, organization_id, business_unit_id) REFERENCES pay_codes(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (submitted_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (reviewed_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_expenses_worker ON driver_expenses(worker_id, organization_id, business_unit_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_expenses_status ON driver_expenses(organization_id, business_unit_id, status)
WHERE
    status = 'Pending';

--bun:split
ALTER TABLE assignments
    ADD COLUMN IF NOT EXISTS ack_status VARCHAR(20) NOT NULL DEFAULT 'Pending',
    ADD COLUMN IF NOT EXISTS ack_at BIGINT,
    ADD COLUMN IF NOT EXISTS ack_reason TEXT;

--bun:split
ALTER TABLE assignments
    ADD CONSTRAINT chk_assignments_ack_status CHECK (ack_status IN ('Pending', 'Accepted', 'Declined'));
