CREATE TYPE "document_preview_status_enum" AS ENUM (
    'Pending',
    'Ready',
    'Failed',
    'Unsupported'
);

ALTER TABLE "documents"
    ADD COLUMN "preview_status" document_preview_status_enum NOT NULL DEFAULT 'Unsupported';

UPDATE "documents"
SET "preview_status" = CASE
    WHEN COALESCE("preview_storage_path", '') <> '' THEN 'Ready'::document_preview_status_enum
    WHEN lower("file_type") LIKE 'image/%' OR lower("file_type") = 'application/pdf' THEN 'Pending'::document_preview_status_enum
    ELSE 'Unsupported'::document_preview_status_enum
END;

CREATE INDEX IF NOT EXISTS "idx_documents_preview_status" ON "documents"("preview_status");
