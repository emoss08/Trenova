-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260409120000_invoice_adjustment_document_support.tx.up.sql

CREATE TABLE IF NOT EXISTS invoice_adjustment_document_references(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "adjustment_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "selected_by_id" TEXT,
    "selected_at" INTEGER,
    "snapshot_file_name" TEXT NOT NULL,
    "snapshot_original_name" TEXT NOT NULL,
    "snapshot_file_type" TEXT NOT NULL,
    "snapshot_resource_type" TEXT NOT NULL,
    "snapshot_resource_id" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_document_references PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_document_references UNIQUE (organization_id, business_unit_id, adjustment_id, document_id),
    CONSTRAINT fk_invoice_adjustment_document_references_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_document_references_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_document_references_adjustment
    ON invoice_adjustment_document_references (organization_id, business_unit_id, adjustment_id);
