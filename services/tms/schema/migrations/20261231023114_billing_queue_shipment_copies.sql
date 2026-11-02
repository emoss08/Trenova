-- A billing queue item stores its shipment's customer, delivery date and pickup date so the queue
-- filters by them from indexes on the item, instead of reading every shipment the customer ever
-- had, or every shipment delivered in the window, and sorting the joined items. A trigger sets them
-- from the shipment whenever an item is written, and a trigger on shipments moves them to the
-- shipment's items when they change, so every writer (the queue, imports, seeds, fixtures) keeps
-- them equal to the shipment's; existing rows are derived here.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "shipment_customer_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "shipment_delivered_at" bigint,
    ADD COLUMN IF NOT EXISTS "shipment_shipped_at" bigint;

UPDATE "billing_queue_items" AS bqi
SET "shipment_customer_id" = s."customer_id",
    "shipment_delivered_at" = s."actual_delivery_date",
    "shipment_shipped_at" = s."actual_ship_date"
FROM "shipments" AS s
WHERE s."id" = bqi."shipment_id"
    AND s."organization_id" = bqi."organization_id"
    AND s."business_unit_id" = bqi."business_unit_id";

CREATE OR REPLACE FUNCTION "billing_queue_items_shipment_copies"()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW."shipment_customer_id" := NULL;
    NEW."shipment_delivered_at" := NULL;
    NEW."shipment_shipped_at" := NULL;
    SELECT s."customer_id", s."actual_delivery_date", s."actual_ship_date"
    INTO NEW."shipment_customer_id", NEW."shipment_delivered_at", NEW."shipment_shipped_at"
    FROM "shipments" AS s
    WHERE s."id" = NEW."shipment_id"
        AND s."organization_id" = NEW."organization_id"
        AND s."business_unit_id" = NEW."business_unit_id";
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_billing_queue_items_shipment_copies" ON "billing_queue_items";

CREATE TRIGGER "trg_billing_queue_items_shipment_copies"
    BEFORE INSERT OR UPDATE OF "shipment_id" ON "billing_queue_items"
    FOR EACH ROW
    EXECUTE FUNCTION "billing_queue_items_shipment_copies"();

CREATE OR REPLACE FUNCTION "shipments_billing_queue_copies"()
    RETURNS TRIGGER
    AS $$
BEGIN
    UPDATE "billing_queue_items"
    SET "shipment_customer_id" = NEW."customer_id",
        "shipment_delivered_at" = NEW."actual_delivery_date",
        "shipment_shipped_at" = NEW."actual_ship_date"
    WHERE "shipment_id" = NEW."id"
        AND "organization_id" = NEW."organization_id"
        AND "business_unit_id" = NEW."business_unit_id";
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_shipments_billing_queue_copies" ON "shipments";

CREATE TRIGGER "trg_shipments_billing_queue_copies"
    AFTER UPDATE OF "customer_id", "actual_delivery_date", "actual_ship_date" ON "shipments"
    FOR EACH ROW
    WHEN (OLD."customer_id" IS DISTINCT FROM NEW."customer_id"
        OR OLD."actual_delivery_date" IS DISTINCT FROM NEW."actual_delivery_date"
        OR OLD."actual_ship_date" IS DISTINCT FROM NEW."actual_ship_date")
    EXECUTE FUNCTION "shipments_billing_queue_copies"();
