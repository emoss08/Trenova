--keel:notx
-- The missing-POD queue pages delivered shipments by their last delivery's arrival inside a window:
-- the index gives that order over the delivered statuses, so the page is read in order and stops at
-- its limit instead of sorting every delivered shipment's stops.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_delivered_arrival" ON "shipments"("organization_id", "business_unit_id", "last_delivery_arrival_at", "id")
WHERE
    "status" IN ('PartiallyCompleted', 'Completed', 'ReadyToInvoice') AND "last_delivery_arrival_at" IS NOT NULL;

--keel:split
-- The unassigned counter counts and totals open shipments without a live assignment by when they
-- were created: the index holds only those shipments, with the charge it totals.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_open_unassigned" ON "shipments"("organization_id", "business_unit_id", "created_at") INCLUDE ("total_charge_amount")
WHERE
    "status" IN ('New', 'PartiallyAssigned', 'Assigned', 'InTransit', 'Delayed', 'PartiallyCompleted') AND "has_active_assignment" IS FALSE;
