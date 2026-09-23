-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003600_assistant_message_kind.tx.up.sql

ALTER TABLE "assistant_messages" ADD COLUMN "kind" TEXT NOT NULL DEFAULT 'Message';
