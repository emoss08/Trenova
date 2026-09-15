-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite cannot drop
-- these columns, so only the indexes are undone. Hand-edits are preserved
-- only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261223000500_invoice_edi_send.tx.down.sql

DROP INDEX IF EXISTS "idx_edi_messages_invoice";

--bun:split

DROP INDEX IF EXISTS "idx_invoices_edi_send_status";
