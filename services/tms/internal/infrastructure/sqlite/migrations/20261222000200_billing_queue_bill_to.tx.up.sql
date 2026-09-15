-- Hand-completed mirror of 20261222000200_billing_queue_bill_to.tx.up.sql.
-- The converter drops aliased UPDATE ... FROM statements, the DO block and the
-- SET NOT NULL, so the backfill is written as correlated subqueries and the
-- column is created NOT NULL with a placeholder default instead.

ALTER TABLE "billing_queue_items" ADD COLUMN "bill_to_customer_id" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "allocated_total_amount" REAL NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "allocated_total_amount_minor" INTEGER NOT NULL DEFAULT 0;

--bun:split

UPDATE "billing_queue_items"
SET "bill_to_customer_id" = (
        SELECT sp."customer_id" FROM "shipments" sp
        WHERE sp."id" = "billing_queue_items"."shipment_id"
          AND sp."organization_id" = "billing_queue_items"."organization_id"
          AND sp."business_unit_id" = "billing_queue_items"."business_unit_id"
    ),
    "allocated_total_amount" = COALESCE((
        SELECT sp."total_charge_amount" FROM "shipments" sp
        WHERE sp."id" = "billing_queue_items"."shipment_id"
          AND sp."organization_id" = "billing_queue_items"."organization_id"
          AND sp."business_unit_id" = "billing_queue_items"."business_unit_id"
    ), 0),
    "allocated_total_amount_minor" = CAST(ROUND(COALESCE((
        SELECT sp."total_charge_amount" FROM "shipments" sp
        WHERE sp."id" = "billing_queue_items"."shipment_id"
          AND sp."organization_id" = "billing_queue_items"."organization_id"
          AND sp."business_unit_id" = "billing_queue_items"."business_unit_id"
    ), 0) * 100) AS INTEGER)
WHERE "bill_to_customer_id" = ''
  AND "shipment_id" IS NOT NULL
  AND EXISTS (
        SELECT 1 FROM "shipments" sp
        WHERE sp."id" = "billing_queue_items"."shipment_id"
          AND sp."organization_id" = "billing_queue_items"."organization_id"
          AND sp."business_unit_id" = "billing_queue_items"."business_unit_id"
  );

--bun:split

UPDATE "billing_queue_items"
SET "bill_to_customer_id" = (
        SELECT inv."customer_id" FROM "invoices" inv
        WHERE inv."id" = "billing_queue_items"."source_invoice_id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
    ),
    "allocated_total_amount" = COALESCE((
        SELECT ABS(inv."total_amount") FROM "invoices" inv
        WHERE inv."id" = "billing_queue_items"."source_invoice_id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
    ), 0),
    "allocated_total_amount_minor" = COALESCE((
        SELECT ABS(inv."total_amount_minor") FROM "invoices" inv
        WHERE inv."id" = "billing_queue_items"."source_invoice_id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
    ), 0)
WHERE "bill_to_customer_id" = ''
  AND "source_invoice_id" IS NOT NULL
  AND EXISTS (
        SELECT 1 FROM "invoices" inv
        WHERE inv."id" = "billing_queue_items"."source_invoice_id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
  );

--bun:split

UPDATE "billing_queue_items"
SET "bill_to_customer_id" = (
        SELECT inv."customer_id" FROM "invoices" inv
        WHERE inv."billing_queue_item_id" = "billing_queue_items"."id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
        LIMIT 1
    ),
    "allocated_total_amount" = COALESCE((
        SELECT ABS(inv."total_amount") FROM "invoices" inv
        WHERE inv."billing_queue_item_id" = "billing_queue_items"."id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
        LIMIT 1
    ), 0),
    "allocated_total_amount_minor" = COALESCE((
        SELECT ABS(inv."total_amount_minor") FROM "invoices" inv
        WHERE inv."billing_queue_item_id" = "billing_queue_items"."id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
        LIMIT 1
    ), 0)
WHERE "bill_to_customer_id" = ''
  AND EXISTS (
        SELECT 1 FROM "invoices" inv
        WHERE inv."billing_queue_item_id" = "billing_queue_items"."id"
          AND inv."organization_id" = "billing_queue_items"."organization_id"
          AND inv."business_unit_id" = "billing_queue_items"."business_unit_id"
  );

--bun:split

UPDATE "billing_queue_items"
SET "bill_to_customer_id" = (
        SELECT o."customer_id" FROM "orders" o
        WHERE o."id" = "billing_queue_items"."order_id"
          AND o."organization_id" = "billing_queue_items"."organization_id"
          AND o."business_unit_id" = "billing_queue_items"."business_unit_id"
    )
WHERE "bill_to_customer_id" = ''
  AND "order_id" IS NOT NULL
  AND EXISTS (
        SELECT 1 FROM "orders" o
        WHERE o."id" = "billing_queue_items"."order_id"
          AND o."organization_id" = "billing_queue_items"."organization_id"
          AND o."business_unit_id" = "billing_queue_items"."business_unit_id"
  );

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_queue_items_bill_to" ON "billing_queue_items" ("bill_to_customer_id", "organization_id", "business_unit_id", "status");
