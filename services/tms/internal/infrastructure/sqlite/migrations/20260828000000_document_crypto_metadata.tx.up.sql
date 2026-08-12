-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260828000000_document_crypto_metadata.tx.up.sql

CREATE INDEX IF NOT EXISTS "idx_documents_crypto_mode"
    ON "documents" ("business_unit_id", "organization_id", "crypto_mode");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_crypto_mode"
    ON "document_upload_sessions" ("business_unit_id", "organization_id", "crypto_mode");
