CREATE TABLE IF NOT EXISTS bank_receipt_import_batches(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    source VARCHAR(100) NOT NULL,
    reference VARCHAR(100),
    status VARCHAR(50) NOT NULL,
    imported_count BIGINT NOT NULL DEFAULT 0,
    matched_count BIGINT NOT NULL DEFAULT 0,
    exception_count BIGINT NOT NULL DEFAULT 0,
    imported_amount_minor BIGINT NOT NULL DEFAULT 0,
    matched_amount_minor BIGINT NOT NULL DEFAULT 0,
    exception_amount_minor BIGINT NOT NULL DEFAULT 0,
    created_by_id VARCHAR(100) NOT NULL,
    updated_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_bank_receipt_import_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipt_import_batches_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipt_import_batches_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
ALTER TABLE bank_receipts ADD COLUMN IF NOT EXISTS import_batch_id VARCHAR(100);

--bun:split
ALTER TABLE bank_receipts
    ADD CONSTRAINT fk_bank_receipts_import_batch
    FOREIGN KEY (import_batch_id, organization_id, business_unit_id)
    REFERENCES bank_receipt_import_batches(id, organization_id, business_unit_id)
    ON DELETE SET NULL;
