-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260408223000_invoice_adjustment_batch_hardening.tx.up.sql

UPDATE billing_queue_items
SET rerate_variance_percent = 0
WHERE rerate_variance_percent IS NULL;
