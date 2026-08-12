-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260404103000_add_shipment_import_chat_status_metadata.tx.up.sql

ALTER TABLE "shipment_import_chat_conversations" ADD COLUMN "status_reason" TEXT;

--bun:split

ALTER TABLE "shipment_import_chat_turns" ADD COLUMN "result_status" TEXT NOT NULL DEFAULT 'Completed';

--bun:split

ALTER TABLE "shipment_import_chat_turns" ADD COLUMN "error_message" TEXT;
