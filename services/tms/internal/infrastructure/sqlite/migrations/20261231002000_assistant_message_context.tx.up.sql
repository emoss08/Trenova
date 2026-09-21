-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002000_assistant_message_context.tx.up.sql

ALTER TABLE "assistant_messages" ADD COLUMN "attachments" TEXT;

--bun:split

ALTER TABLE "assistant_messages" ADD COLUMN "mentions" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_threads_origin_last_message" ON "assistant_threads" ("origin", "last_message_at");
