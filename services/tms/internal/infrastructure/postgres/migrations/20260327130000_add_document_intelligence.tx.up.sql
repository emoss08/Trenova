CREATE TYPE "document_content_status_enum" AS ENUM (
    'Pending',
    'Extracting',
    'Extracted',
    'Indexed',
    'Failed'
);

CREATE TYPE "document_shipment_draft_status_enum" AS ENUM (
    'Unavailable',
    'Pending',
    'Ready',
    'Failed'
);

ALTER TABLE "documents"
    ADD COLUMN "content_status" document_content_status_enum NOT NULL DEFAULT 'Pending',
    ADD COLUMN "content_error" TEXT,
    ADD COLUMN "detected_kind" VARCHAR(100),
    ADD COLUMN "has_extracted_text" BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN "shipment_draft_status" document_shipment_draft_status_enum NOT NULL DEFAULT 'Unavailable';

CREATE INDEX IF NOT EXISTS "idx_documents_content_status" ON "documents"("content_status");
CREATE INDEX IF NOT EXISTS "idx_documents_detected_kind" ON "documents"("detected_kind");

CREATE TABLE IF NOT EXISTS "document_contents" (
    "id" VARCHAR(100) NOT NULL,
    "document_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "status" document_content_status_enum NOT NULL DEFAULT 'Pending',
    "content_text" TEXT,
    "page_count" INTEGER NOT NULL DEFAULT 0,
    "source_kind" VARCHAR(20),
    "detected_language" VARCHAR(20),
    "detected_document_kind" VARCHAR(100),
    "classification_confidence" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "structured_data" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "failure_code" VARCHAR(100),
    "failure_message" TEXT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "last_extracted_at" BIGINT,
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("content_text", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("detected_document_kind", '')), 'B')
    ) STORED,
    CONSTRAINT "pk_document_contents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_document_contents_document" UNIQUE ("document_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_contents_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_document_contents_document" ON "document_contents"("document_id", "organization_id", "business_unit_id");
CREATE INDEX IF NOT EXISTS "idx_document_contents_status" ON "document_contents"("status");
CREATE INDEX IF NOT EXISTS "idx_document_contents_search_vector" ON "document_contents" USING GIN("search_vector");

CREATE TABLE IF NOT EXISTS "document_shipment_drafts" (
    "id" VARCHAR(100) NOT NULL,
    "document_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "status" document_shipment_draft_status_enum NOT NULL DEFAULT 'Unavailable',
    "document_kind" VARCHAR(100),
    "confidence" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "draft_data" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "failure_code" VARCHAR(100),
    "failure_message" TEXT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_shipment_drafts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_document_shipment_drafts_document" UNIQUE ("document_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_shipment_drafts_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_document_shipment_drafts_document" ON "document_shipment_drafts"("document_id", "organization_id", "business_unit_id");
CREATE INDEX IF NOT EXISTS "idx_document_shipment_drafts_status" ON "document_shipment_drafts"("status");
