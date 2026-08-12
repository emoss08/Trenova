-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260401120000_add_document_shipment_draft_attachment_metadata.tx.up.sql

ALTER TABLE "document_shipment_drafts" ADD COLUMN "attached_shipment_id" TEXT;

--bun:split

ALTER TABLE "document_shipment_drafts" ADD COLUMN "attached_at" INTEGER;

--bun:split

ALTER TABLE "document_shipment_drafts" ADD COLUMN "attached_by_id" TEXT;
