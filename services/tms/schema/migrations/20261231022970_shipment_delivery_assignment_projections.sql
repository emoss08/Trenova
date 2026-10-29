-- Two values read from a shipment's moves, stops and assignments are stored on the shipment so the
-- reads that filter and sort by them reach an index: the actual arrival of its last delivered stop
-- (the last non-canceled delivery stop, by move and stop sequence, on a non-canceled move, that has
-- an actual arrival), which orders the missing-POD queue, and whether a non-canceled move has a
-- live (not archived, not canceled) assignment, which the unassigned and ready-to-dispatch
-- counters split open shipments by. shipmentprojection recomputes both in the transaction of every
-- write that changes a stop, a move or an assignment; existing rows are derived here.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "last_delivery_arrival_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "has_active_assignment" BOOLEAN NOT NULL DEFAULT false;

UPDATE "shipments" AS sp
SET "last_delivery_arrival_at" = delivered."actual_arrival"
FROM (
    SELECT DISTINCT ON (sm."shipment_id", sm."organization_id", sm."business_unit_id")
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        st."actual_arrival"
    FROM "stops" AS st
    JOIN "shipment_moves" AS sm
        ON sm."id" = st."shipment_move_id"
        AND sm."organization_id" = st."organization_id"
        AND sm."business_unit_id" = st."business_unit_id"
    WHERE sm."status" <> 'Canceled'
        AND st."type" IN ('Delivery', 'SplitDelivery')
        AND st."status" <> 'Canceled'
        AND st."actual_arrival" IS NOT NULL
    ORDER BY
        sm."shipment_id",
        sm."organization_id",
        sm."business_unit_id",
        sm."sequence" DESC,
        st."sequence" DESC
) AS delivered
WHERE sp."id" = delivered."shipment_id"
    AND sp."organization_id" = delivered."organization_id"
    AND sp."business_unit_id" = delivered."business_unit_id";

UPDATE "shipments" AS sp
SET "has_active_assignment" = true
WHERE EXISTS (
    SELECT 1
    FROM "shipment_moves" AS sm
    JOIN "assignments" AS a
        ON a."shipment_move_id" = sm."id"
        AND a."organization_id" = sm."organization_id"
        AND a."business_unit_id" = sm."business_unit_id"
    WHERE sm."shipment_id" = sp."id"
        AND sm."organization_id" = sp."organization_id"
        AND sm."business_unit_id" = sp."business_unit_id"
        AND sm."status" <> 'Canceled'
        AND a."archived_at" IS NULL
        AND a."status" <> 'Canceled'
);
