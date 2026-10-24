ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "crypto_mode" varchar(32) NOT NULL DEFAULT 'envelope_v1',
    ADD COLUMN IF NOT EXISTS "crypto_version" smallint NOT NULL DEFAULT 1;

ALTER TABLE "document_upload_sessions"
    ADD COLUMN IF NOT EXISTS "checksum_sha256" varchar(64),
    ADD COLUMN IF NOT EXISTS "crypto_mode" varchar(32) NOT NULL DEFAULT 'envelope_v1',
    ADD COLUMN IF NOT EXISTS "crypto_version" smallint NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS "idx_documents_crypto_mode"
    ON "documents"("business_unit_id", "organization_id", "crypto_mode");

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_crypto_mode"
    ON "document_upload_sessions"("business_unit_id", "organization_id", "crypto_mode");
