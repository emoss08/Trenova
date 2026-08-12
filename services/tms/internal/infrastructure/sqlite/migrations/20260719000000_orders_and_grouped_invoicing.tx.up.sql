-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260719000000_orders_and_grouped_invoicing.tx.up.sql

CREATE TABLE IF NOT EXISTS "orders"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "owner_id" TEXT,
    "entered_by_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "order_number" TEXT NOT NULL,
    "po_number" TEXT,
    "bol" TEXT,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "quoted_amount" NUMERIC,
    "base_amount" NUMERIC,
    "total_amount" NUMERIC NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_orders" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_orders_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_orders_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_orders_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_orders_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_orders_entered_by" FOREIGN KEY ("entered_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_orders_order_number" ON "orders" (lower("order_number"), "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_orders_business_unit" ON "orders" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_orders_organization" ON "orders" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_orders_customer" ON "orders" ("customer_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_orders_created_updated" ON "orders" ("created_at", "updated_at");

--bun:split

ALTER TABLE "shipments" ADD COLUMN "order_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments ("order_id");

--bun:split

ALTER TABLE "orders" ADD COLUMN "backfill_shipment_id" TEXT;

--bun:split

UPDATE
    "shipments"
SET
    "order_id" = o."id"
FROM
    "orders" o
WHERE
    o."backfill_shipment_id" = "shipments"."id"
    AND o."organization_id" = "shipments"."organization_id"
    AND o."business_unit_id" = "shipments"."business_unit_id";

--bun:split

ALTER TABLE "invoices" ADD COLUMN "order_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "order_number" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoices_order ON invoices ("order_id");

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "order_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_order ON billing_queue_items ("order_id");

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "shipment_id" TEXT;

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "shipment_pro_number" TEXT;

--bun:split

ALTER TABLE "invoice_lines" ADD COLUMN "shipment_bol" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_lines_shipment ON invoice_lines ("shipment_id");
