-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006480_assistant_message_found_tools.tx.up.sql

ALTER TABLE "assistant_messages" ADD COLUMN "found_tools" TEXT;
