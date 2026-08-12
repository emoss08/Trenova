-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327193000_add_document_content_pages.tx.up.sql

CREATE TABLE IF NOT EXISTS document_content_pages (
    "id" TEXT PRIMARY KEY,
    "document_content_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "page_number" INTEGER NOT NULL,
    "source_kind" TEXT NOT NULL,
    "extracted_text" TEXT,
    "ocr_confidence" REAL NOT NULL DEFAULT 0,
    "preprocessing_applied" INTEGER NOT NULL DEFAULT 0,
    "width" INTEGER NOT NULL DEFAULT 0,
    "height" INTEGER NOT NULL DEFAULT 0,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT fk_document_content_pages_content FOREIGN KEY (document_content_id, organization_id, business_unit_id)
        REFERENCES document_contents(id, organization_id, business_unit_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_document_content_pages_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_document_content_pages_page UNIQUE (document_content_id, organization_id, business_unit_id, page_number)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_content_pages_document
    ON document_content_pages (document_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_content_pages_content_page
    ON document_content_pages (document_content_id, organization_id, business_unit_id, page_number);
