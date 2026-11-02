-- When a shipment is picked up and when it delivers are stored on it. pickup_at is the scheduled
-- window start of its first pickup stop (the first non-canceled pickup stop, by move and stop
-- sequence, on a non-canceled move). delivery_at is the actual arrival of its last delivery stop
-- (the last non-canceled delivery stop, by move and stop sequence, on a non-canceled move), or that
-- stop's scheduled window start until it arrives. The pickup-window, delivering-today and
-- delivery-hour quick filters, their counts and the board watchlist compare them instead of
-- searching the stops of every shipment they read. shipmentprojection recomputes both in the
-- transaction of every write that changes a stop or a move; existing rows are derived here.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "pickup_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "delivery_at" BIGINT;

UPDATE "shipments" AS sp
SET "pickup_at" = pickup."pickup_at"
FROM (
    SELECT DISTINCT ON (sm."shipment_id", sm."organization_id", sm."business_unit_id")
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        st."scheduled_window_start" AS "pickup_at"
    FROM "stops" AS st
    JOIN "shipment_moves" AS sm
        ON sm."id" = st."shipment_move_id"
        AND sm."organization_id" = st."organization_id"
        AND sm."business_unit_id" = st."business_unit_id"
    WHERE sm."status" <> 'Canceled'
        AND st."type" IN ('Pickup', 'SplitPickup')
        AND st."status" <> 'Canceled'
    ORDER BY
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        sm."sequence" ASC,
        st."sequence" ASC
) AS pickup
WHERE sp."id" = pickup."shipment_id"
    AND sp."organization_id" = pickup."organization_id"
    AND sp."business_unit_id" = pickup."business_unit_id";

UPDATE "shipments" AS sp
SET "delivery_at" = delivery."delivery_at"
FROM (
    SELECT DISTINCT ON (sm."shipment_id", sm."organization_id", sm."business_unit_id")
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        coalesce(st."actual_arrival", st."scheduled_window_start") AS "delivery_at"
    FROM "stops" AS st
    JOIN "shipment_moves" AS sm
        ON sm."id" = st."shipment_move_id"
        AND sm."organization_id" = st."organization_id"
        AND sm."business_unit_id" = st."business_unit_id"
    WHERE sm."status" <> 'Canceled'
        AND st."type" IN ('Delivery', 'SplitDelivery')
        AND st."status" <> 'Canceled'
    ORDER BY
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        sm."sequence" DESC,
        st."sequence" DESC
) AS delivery
WHERE sp."id" = delivery."shipment_id"
    AND sp."organization_id" = delivery."organization_id"
    AND sp."business_unit_id" = delivery."business_unit_id";
