DROP INDEX IF EXISTS "idx_documents_preview_status";

ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "preview_status";

DROP TYPE IF EXISTS "document_preview_status_enum";
