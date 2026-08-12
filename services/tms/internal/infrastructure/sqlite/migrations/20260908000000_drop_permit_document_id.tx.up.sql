-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260908000000_drop_permit_document_id.tx.up.sql

ALTER TABLE "permits" DROP COLUMN "document_id";
