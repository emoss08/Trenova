CREATE TABLE IF NOT EXISTS invoice_adjustment_document_references(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    adjustment_id VARCHAR(100) NOT NULL,
    document_id VARCHAR(100) NOT NULL,
    selected_by_id VARCHAR(100),
    selected_at BIGINT,
    snapshot_file_name VARCHAR(255) NOT NULL,
    snapshot_original_name VARCHAR(255) NOT NULL,
    snapshot_file_type VARCHAR(100) NOT NULL,
    snapshot_resource_type VARCHAR(100) NOT NULL,
    snapshot_resource_id VARCHAR(100) NOT NULL,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_adjustment_document_references PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_document_references UNIQUE (organization_id, business_unit_id, adjustment_id, document_id),
    CONSTRAINT fk_invoice_adjustment_document_references_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_document_references_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_document_references_adjustment
    ON invoice_adjustment_document_references(organization_id, business_unit_id, adjustment_id);
