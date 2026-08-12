-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260407160506_remove_billing_queue_item_unique_index.up.sql

DROP INDEX IF EXISTS "idx_billing_queue_items_shipment_bill_type";
