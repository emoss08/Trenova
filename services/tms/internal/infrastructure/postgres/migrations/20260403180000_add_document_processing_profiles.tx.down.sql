ALTER TABLE "document_upload_sessions"
    DROP COLUMN IF EXISTS "processing_profile";

ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "processing_profile";
