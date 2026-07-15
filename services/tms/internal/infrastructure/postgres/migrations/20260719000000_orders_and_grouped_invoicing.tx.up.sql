--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Commercial order aggregate: one order groups one or more shipments (legs) under
-- a single customer, owning order-level quote/AR and a derived lifecycle status.

-- Runtime order-number generation uses the "order" sequence type.
ALTER TYPE "sequence_type_enum" ADD VALUE IF NOT EXISTS 'order';

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status_enum') THEN
        CREATE TYPE order_status_enum AS ENUM(
            'Draft',
            'Confirmed',
            'InProgress',
            'Completed',
            'Billed',
            'Closed',
            'Canceled'
        );
    END IF;
END
$$;

--bun:split
CREATE TABLE IF NOT EXISTS "orders"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "owner_id" varchar(100),
    "entered_by_id" varchar(100),
    "status" order_status_enum NOT NULL DEFAULT 'Draft',
    "order_number" varchar(100) NOT NULL,
    "po_number" varchar(100),
    "bol" varchar(100),
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "quoted_amount" numeric(19, 4),
    "base_amount" numeric(19, 4),
    "total_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_orders" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_orders_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_orders_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_orders_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_orders_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_orders_entered_by" FOREIGN KEY ("entered_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX "idx_orders_order_number" ON "orders"(lower("order_number"), "organization_id");

CREATE INDEX "idx_orders_business_unit" ON "orders"("business_unit_id");

CREATE INDEX "idx_orders_organization" ON "orders"("organization_id");

CREATE INDEX "idx_orders_customer" ON "orders"("customer_id");

CREATE INDEX "idx_orders_created_updated" ON "orders"("created_at", "updated_at");

COMMENT ON TABLE "orders" IS 'Commercial order aggregate grouping shipments (legs) under one customer';

--bun:split
ALTER TABLE "orders"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("order_number", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("po_number", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE("bol", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_orders_search ON orders USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_orders_trgm_order_number ON orders USING gin(order_number gin_trgm_ops);

--bun:split
-- Link shipments to their parent order (nullable; standalone shipments have none).
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "order_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_order" FOREIGN KEY ("order_id", "organization_id", "business_unit_id") REFERENCES "orders"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments("order_id");

--bun:split
-- Backfill: one order per existing shipment (faithful 1:1). A temporary column
-- carries the source shipment id so the two statements can be correlated, then is
-- dropped. Backfilled numbers use the distinct 'O-BF-' prefix so they never collide
-- with runtime GenerateOrderNumber output ('O' + date + counter) and do not require
-- advancing the tenant sequence counter.
ALTER TABLE "orders"
    ADD COLUMN IF NOT EXISTS "backfill_shipment_id" varchar(100);

INSERT INTO "orders"("id", "business_unit_id", "organization_id", "customer_id", "status", "order_number", "currency_code", "total_amount", "version", "created_at", "updated_at", "backfill_shipment_id")
SELECT
    CONCAT('ord_', replace(gen_random_uuid()::text, '-', '')),
    s."business_unit_id",
    s."organization_id",
    s."customer_id",
    'Confirmed',
    'O-BF-' || lpad((row_number() OVER (PARTITION BY s."organization_id" ORDER BY s."created_at", s."id"))::text, 8, '0'),
    'USD',
    COALESCE(s."total_charge_amount", 0),
    0,
    s."created_at",
    s."updated_at",
    s."id"
FROM
    "shipments" s
WHERE
    s."order_id" IS NULL;

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

ALTER TABLE "orders"
    DROP COLUMN IF EXISTS "backfill_shipment_id";

--bun:split
-- Grouped invoicing: an invoice/billing-queue item may belong to an order, and each
-- invoice line is attributable to a specific leg.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "order_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "order_number" varchar(100);

ALTER TABLE "invoices"
    ALTER COLUMN "shipment_id" DROP NOT NULL;

ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_order" FOREIGN KEY ("order_id", "organization_id", "business_unit_id") REFERENCES "orders"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_invoices_order ON invoices("order_id");

--bun:split
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "order_id" varchar(100);

ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "fk_billing_queue_items_order" FOREIGN KEY ("order_id", "organization_id", "business_unit_id") REFERENCES "orders"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_order ON billing_queue_items("order_id");

--bun:split
ALTER TABLE "invoice_lines"
    ADD COLUMN IF NOT EXISTS "shipment_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "shipment_pro_number" varchar(100),
    ADD COLUMN IF NOT EXISTS "shipment_bol" varchar(100);

CREATE INDEX IF NOT EXISTS idx_invoice_lines_shipment ON invoice_lines("shipment_id");
