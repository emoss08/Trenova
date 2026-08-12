-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327143000_add_document_search_projections.tx.up.sql

CREATE TABLE IF NOT EXISTS document_search_projections (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "resource_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "original_name" TEXT NOT NULL,
    "description" TEXT,
    "tags" TEXT NOT NULL DEFAULT '{}',
    "status" TEXT NOT NULL,
    "content_status" TEXT NOT NULL DEFAULT 'Pending',
    "detected_kind" TEXT,
    "shipment_draft_status" TEXT NOT NULL DEFAULT 'Unavailable',
    "content_text" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_document_search_projections PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_document_search_projections_document FOREIGN KEY (id, organization_id, business_unit_id)
        REFERENCES documents (id, organization_id, business_unit_id)
        ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_search_projections_resource
    ON document_search_projections (organization_id, business_unit_id, resource_type, resource_id);
