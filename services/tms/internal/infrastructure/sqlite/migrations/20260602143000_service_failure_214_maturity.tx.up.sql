-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260602143000_service_failure_214_maturity.tx.up.sql

ALTER TABLE "edi_messages" ADD COLUMN "ack_status" TEXT;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "ack_message_id" TEXT;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "ack_received_at" INTEGER;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "ack_last_error" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_ack"
    ON "edi_messages" ("organization_id", "business_unit_id", "ack_status", "generated_at" DESC)WHERE "ack_status" IS NOT NULL;
