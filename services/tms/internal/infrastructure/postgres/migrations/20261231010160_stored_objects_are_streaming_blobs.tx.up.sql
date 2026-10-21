-- Documents, upload sessions, capture pages, thumbnails and page renders are
-- sealed as streaming blobs (encryptionservice.SealBlob), recorded as
-- crypto_mode blob_v1, crypto_version 2.
ALTER TABLE "documents"
    ALTER COLUMN "crypto_mode" SET DEFAULT 'blob_v1',
    ALTER COLUMN "crypto_version" SET DEFAULT 2;

--bun:split
ALTER TABLE "document_upload_sessions"
    ALTER COLUMN "crypto_mode" SET DEFAULT 'blob_v1',
    ALTER COLUMN "crypto_version" SET DEFAULT 2;
