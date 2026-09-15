-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261222000400_invoice_split_bill.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "shipper_customer_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "is_split_bill" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_shipper_customer" ON "invoices" ("shipper_customer_id", "organization_id", "business_unit_id")WHERE
    "shipper_customer_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_shipment" ON "invoices" ("shipment_id", "organization_id", "business_unit_id")WHERE
    "shipment_id" IS NOT NULL;

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "allocation_percent" REAL;

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "charge_allocation_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_lines_charge_allocation" ON "invoice_lines" ("charge_allocation_id", "organization_id", "business_unit_id")WHERE
    "charge_allocation_id" IS NOT NULL;
