-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260403180000_add_document_processing_profiles.tx.up.sql

ALTER TABLE "documents" ADD COLUMN "processing_profile" TEXT NOT NULL DEFAULT 'none';

--bun:split

ALTER TABLE "document_upload_sessions" ADD COLUMN "processing_profile" TEXT NOT NULL DEFAULT 'none';
