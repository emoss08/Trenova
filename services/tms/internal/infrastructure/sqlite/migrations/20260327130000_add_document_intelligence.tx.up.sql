-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327130000_add_document_intelligence.tx.up.sql

ALTER TABLE "documents" ADD COLUMN "content_status" TEXT NOT NULL DEFAULT 'Pending';

--bun:split

ALTER TABLE "documents" ADD COLUMN "content_error" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "detected_kind" TEXT;

--bun:split

ALTER TABLE "documents" ADD COLUMN "has_extracted_text" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "documents" ADD COLUMN "shipment_draft_status" TEXT NOT NULL DEFAULT 'Unavailable';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_content_status" ON "documents" ("content_status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_detected_kind" ON "documents" ("detected_kind");

--bun:split

CREATE TABLE IF NOT EXISTS "document_contents" (
    "id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "content_text" TEXT,
    "page_count" INTEGER NOT NULL DEFAULT 0,
    "source_kind" TEXT,
    "detected_language" TEXT,
    "detected_document_kind" TEXT,
    "classification_confidence" REAL NOT NULL DEFAULT 0,
    "structured_data" TEXT NOT NULL DEFAULT '{}',
    "failure_code" TEXT,
    "failure_message" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "last_extracted_at" INTEGER,
    CONSTRAINT "pk_document_contents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_document_contents_document" UNIQUE ("document_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_contents_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_contents_document" ON "document_contents" ("document_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_contents_status" ON "document_contents" ("status");

--bun:split

CREATE TABLE IF NOT EXISTS "document_shipment_drafts" (
    "id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Unavailable',
    "document_kind" TEXT,
    "confidence" REAL NOT NULL DEFAULT 0,
    "draft_data" TEXT NOT NULL DEFAULT '{}',
    "failure_code" TEXT,
    "failure_message" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_shipment_drafts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_document_shipment_drafts_document" UNIQUE ("document_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_shipment_drafts_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_shipment_drafts_document" ON "document_shipment_drafts" ("document_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_shipment_drafts_status" ON "document_shipment_drafts" ("status");
