-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260212000001_add_document_type_is_system.tx.up.sql

ALTER TABLE "document_types" ADD COLUMN "is_system" INTEGER NOT NULL DEFAULT 0;
