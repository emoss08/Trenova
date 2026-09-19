-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000200_document_ai_extraction_provider.tx.up.sql

ALTER TABLE "document_ai_extractions" ADD COLUMN "provider_id" TEXT;
