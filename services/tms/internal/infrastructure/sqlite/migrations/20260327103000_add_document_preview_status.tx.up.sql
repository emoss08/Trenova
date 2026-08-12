-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327103000_add_document_preview_status.tx.up.sql

ALTER TABLE "documents" ADD COLUMN "preview_status" TEXT NOT NULL DEFAULT 'Unsupported';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_preview_status" ON "documents" ("preview_status");
