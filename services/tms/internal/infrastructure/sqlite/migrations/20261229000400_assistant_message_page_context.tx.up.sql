-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261229000400_assistant_message_page_context.tx.up.sql

ALTER TABLE "assistant_messages" ADD COLUMN "page_context" TEXT;
