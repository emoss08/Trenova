CREATE TABLE IF NOT EXISTS bank_receipt_work_items(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    bank_receipt_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    assigned_to_user_id VARCHAR(100),
    assigned_at BIGINT,
    resolution_type VARCHAR(50),
    resolution_note TEXT,
    resolved_by_user_id VARCHAR(100),
    resolved_at BIGINT,
    created_by_id VARCHAR(100) NOT NULL,
    updated_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_bank_receipt_work_items PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipt_work_items_receipt FOREIGN KEY (bank_receipt_id, organization_id, business_unit_id) REFERENCES bank_receipts(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_bank_receipt_work_items_assigned_to FOREIGN KEY (assigned_to_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipt_work_items_resolved_by FOREIGN KEY (resolved_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipt_work_items_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipt_work_items_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_receipt_work_items_active
    ON bank_receipt_work_items(organization_id, business_unit_id, bank_receipt_id)
    WHERE status IN ('Open', 'Assigned', 'InReview');
