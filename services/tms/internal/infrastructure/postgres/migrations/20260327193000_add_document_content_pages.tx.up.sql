CREATE TABLE IF NOT EXISTS document_content_pages (
    id VARCHAR(100) PRIMARY KEY,
    document_content_id VARCHAR(100) NOT NULL,
    document_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    page_number INTEGER NOT NULL,
    source_kind VARCHAR(20) NOT NULL,
    extracted_text TEXT,
    ocr_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    preprocessing_applied BOOLEAN NOT NULL DEFAULT FALSE,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT fk_document_content_pages_content FOREIGN KEY (document_content_id, organization_id, business_unit_id)
        REFERENCES document_contents(id, organization_id, business_unit_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_document_content_pages_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_document_content_pages_page UNIQUE (document_content_id, organization_id, business_unit_id, page_number)
);

CREATE INDEX IF NOT EXISTS idx_document_content_pages_document
    ON document_content_pages (document_id, organization_id, business_unit_id);

CREATE INDEX IF NOT EXISTS idx_document_content_pages_content_page
    ON document_content_pages (document_content_id, organization_id, business_unit_id, page_number);
