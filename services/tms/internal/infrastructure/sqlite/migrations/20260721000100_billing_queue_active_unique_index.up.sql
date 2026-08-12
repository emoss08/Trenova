-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260721000100_billing_queue_active_unique_index.up.sql

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_queue_items_active_shipment_bill_type" ON "billing_queue_items" ("shipment_id", "organization_id", "business_unit_id", "bill_type")WHERE
    "status" NOT IN ('Posted', 'Canceled');
