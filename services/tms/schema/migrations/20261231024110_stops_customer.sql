-- A stop carries its shipment's customer, so stop service can be added up by customer from the
-- stops alone (stop_service_customer_daily). The database keeps it: the application never writes
-- the column. A stop takes its customer when it is inserted or moved to another move, a move's
-- stops follow it to another shipment, and a shipment's stops follow a change of its customer.
ALTER TABLE "stops" ADD COLUMN IF NOT EXISTS "customer_id" varchar(100);

UPDATE "stops" s
SET "customer_id" = sp."customer_id"
FROM "shipment_moves" sm
JOIN "shipments" sp
    ON sp."id" = sm."shipment_id"
    AND sp."organization_id" = sm."organization_id"
    AND sp."business_unit_id" = sm."business_unit_id"
WHERE sm."id" = s."shipment_move_id"
    AND sm."organization_id" = s."organization_id"
    AND sm."business_unit_id" = s."business_unit_id"
    AND s."customer_id" IS DISTINCT FROM sp."customer_id";

CREATE OR REPLACE FUNCTION trenova_stop_customer()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    SELECT sp."customer_id" INTO NEW."customer_id"
    FROM "shipment_moves" sm
    JOIN "shipments" sp
        ON sp."id" = sm."shipment_id"
        AND sp."organization_id" = sm."organization_id"
        AND sp."business_unit_id" = sm."business_unit_id"
    WHERE sm."id" = NEW."shipment_move_id"
        AND sm."organization_id" = NEW."organization_id"
        AND sm."business_unit_id" = NEW."business_unit_id";
    RETURN NEW;
END
$$;

CREATE OR REPLACE TRIGGER "trg_stops_customer"
    BEFORE INSERT OR UPDATE OF "shipment_move_id", "customer_id" ON "stops"
    FOR EACH ROW
    EXECUTE FUNCTION trenova_stop_customer();

CREATE OR REPLACE FUNCTION trenova_move_stops_customer()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE "stops" s
    SET "customer_id" = NULL
    WHERE s."shipment_move_id" = NEW."id"
        AND s."organization_id" = NEW."organization_id"
        AND s."business_unit_id" = NEW."business_unit_id";
    RETURN NULL;
END
$$;

CREATE OR REPLACE TRIGGER "trg_shipment_moves_stops_customer"
    AFTER UPDATE OF "shipment_id" ON "shipment_moves"
    FOR EACH ROW
    WHEN (OLD."shipment_id" IS DISTINCT FROM NEW."shipment_id")
    EXECUTE FUNCTION trenova_move_stops_customer();

CREATE OR REPLACE FUNCTION trenova_shipment_stops_customer()
    RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE "stops" s
    SET "customer_id" = NEW."customer_id"
    FROM "shipment_moves" sm
    WHERE sm."shipment_id" = NEW."id"
        AND sm."organization_id" = NEW."organization_id"
        AND sm."business_unit_id" = NEW."business_unit_id"
        AND s."shipment_move_id" = sm."id"
        AND s."organization_id" = sm."organization_id"
        AND s."business_unit_id" = sm."business_unit_id";
    RETURN NULL;
END
$$;

CREATE OR REPLACE TRIGGER "trg_shipments_stops_customer"
    AFTER UPDATE OF "customer_id" ON "shipments"
    FOR EACH ROW
    WHEN (OLD."customer_id" IS DISTINCT FROM NEW."customer_id")
    EXECUTE FUNCTION trenova_shipment_stops_customer();
