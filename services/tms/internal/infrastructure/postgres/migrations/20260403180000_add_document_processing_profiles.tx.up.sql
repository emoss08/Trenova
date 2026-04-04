ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "processing_profile" VARCHAR(64) NOT NULL DEFAULT 'none';

ALTER TABLE "document_upload_sessions"
    ADD COLUMN IF NOT EXISTS "processing_profile" VARCHAR(64) NOT NULL DEFAULT 'none';
