-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260327213000_add_ai_document_intelligence_controls.tx.up.sql

ALTER TABLE "document_controls" ADD COLUMN "enable_ai_assisted_classification" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "document_controls" ADD COLUMN "enable_ai_assisted_extraction" INTEGER NOT NULL DEFAULT 0;
