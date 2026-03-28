DROP TABLE IF EXISTS "document_shipment_drafts";
DROP TABLE IF EXISTS "document_contents";

DROP INDEX IF EXISTS "idx_documents_detected_kind";
DROP INDEX IF EXISTS "idx_documents_content_status";

ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "shipment_draft_status",
    DROP COLUMN IF EXISTS "has_extracted_text",
    DROP COLUMN IF EXISTS "detected_kind",
    DROP COLUMN IF EXISTS "content_error",
    DROP COLUMN IF EXISTS "content_status";

DROP TYPE IF EXISTS "document_shipment_draft_status_enum";
DROP TYPE IF EXISTS "document_content_status_enum";
