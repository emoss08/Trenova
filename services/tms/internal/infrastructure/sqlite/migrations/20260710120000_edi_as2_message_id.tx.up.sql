-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260710120000_edi_as2_message_id.tx.up.sql

ALTER TABLE "edi_messages" ADD COLUMN "as2_message_id" TEXT;

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "as2_mic" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_as2_message_id" ON "edi_messages" ("as2_message_id")WHERE
    "as2_message_id" IS NOT NULL;
