-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261222000000_shipment_bill_to.tx.up.sql

ALTER TABLE "shipments" ADD COLUMN "bill_to_customer_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "freight_terms" TEXT NOT NULL DEFAULT 'Prepaid';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_bill_to_customer" ON "shipments" ("bill_to_customer_id", "organization_id", "business_unit_id")WHERE
    "bill_to_customer_id" IS NOT NULL;
