CREATE TABLE IF NOT EXISTS document_search_projections (
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    description TEXT,
    tags VARCHAR(100)[] NOT NULL DEFAULT '{}'::VARCHAR(100)[],
    status VARCHAR(100) NOT NULL,
    content_status VARCHAR(100) NOT NULL DEFAULT 'Pending',
    detected_kind VARCHAR(100),
    shipment_draft_status VARCHAR(100) NOT NULL DEFAULT 'Unavailable',
    content_text TEXT,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_document_search_projections PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_document_search_projections_document FOREIGN KEY (id, organization_id, business_unit_id)
        REFERENCES documents (id, organization_id, business_unit_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_search_projections_resource
    ON document_search_projections (organization_id, business_unit_id, resource_type, resource_id);

INSERT INTO document_search_projections (
    id,
    organization_id,
    business_unit_id,
    resource_id,
    resource_type,
    file_name,
    original_name,
    description,
    tags,
    status,
    content_status,
    detected_kind,
    shipment_draft_status,
    content_text,
    created_at,
    updated_at
)
SELECT
    doc.id,
    doc.organization_id,
    doc.business_unit_id,
    doc.resource_id,
    doc.resource_type,
    doc.file_name,
    doc.original_name,
    doc.description,
    COALESCE(doc.tags, '{}'::VARCHAR(100)[]),
    doc.status::TEXT,
    COALESCE(doc.content_status::TEXT, 'Pending'),
    NULLIF(doc.detected_kind, ''),
    COALESCE(doc.shipment_draft_status::TEXT, 'Unavailable'),
    dc.content_text,
    doc.created_at,
    doc.updated_at
FROM documents AS doc
LEFT JOIN document_contents AS dc
    ON dc.document_id = doc.id
    AND dc.organization_id = doc.organization_id
    AND dc.business_unit_id = doc.business_unit_id
ON CONFLICT (id, organization_id, business_unit_id) DO UPDATE
SET resource_id = EXCLUDED.resource_id,
    resource_type = EXCLUDED.resource_type,
    file_name = EXCLUDED.file_name,
    original_name = EXCLUDED.original_name,
    description = EXCLUDED.description,
    tags = EXCLUDED.tags,
    status = EXCLUDED.status,
    content_status = EXCLUDED.content_status,
    detected_kind = EXCLUDED.detected_kind,
    shipment_draft_status = EXCLUDED.shipment_draft_status,
    content_text = EXCLUDED.content_text,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at;
