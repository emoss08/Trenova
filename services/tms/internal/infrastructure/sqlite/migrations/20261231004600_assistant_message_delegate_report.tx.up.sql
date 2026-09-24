-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231004600_assistant_message_delegate_report.tx.up.sql

ALTER TABLE "assistant_messages" ADD COLUMN "delegate_report" TEXT;
