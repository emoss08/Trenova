ALTER TABLE "documents"
    ALTER COLUMN "crypto_mode" SET DEFAULT 'envelope_v1',
    ALTER COLUMN "crypto_version" SET DEFAULT 1;

--bun:split
ALTER TABLE "document_upload_sessions"
    ALTER COLUMN "crypto_mode" SET DEFAULT 'envelope_v1',
    ALTER COLUMN "crypto_version" SET DEFAULT 1;
