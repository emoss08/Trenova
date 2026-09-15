-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed. Hand-edits are
-- preserved only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261223000200_standalone_memos.tx.down.sql

DROP TABLE IF EXISTS "credit_memo_applications";

--bun:split

DROP INDEX IF EXISTS "idx_invoices_reference_invoice";
