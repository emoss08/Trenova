DROP INDEX IF EXISTS "idx_document_upload_sessions_crypto_mode";
DROP INDEX IF EXISTS "idx_documents_crypto_mode";

ALTER TABLE "document_upload_sessions"
    DROP COLUMN IF EXISTS "crypto_version",
    DROP COLUMN IF EXISTS "crypto_mode",
    DROP COLUMN IF EXISTS "checksum_sha256";

ALTER TABLE "documents"
    DROP COLUMN IF EXISTS "crypto_version",
    DROP COLUMN IF EXISTS "crypto_mode";
