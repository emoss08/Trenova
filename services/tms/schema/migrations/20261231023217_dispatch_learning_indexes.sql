--keel:notx
-- Learning reads the driver assignments people made after its cursor, in the order
-- they happened, and lists the organizations that learn their dispatch style.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipment_events_user_driver_assigned"
    ON "shipment_events"("organization_id", "business_unit_id", "occurred_at", "id")
    INCLUDE ("move_id", "assignment_id")
    WHERE "type" = 'DriverAssigned' AND "actor_type" = 'user';

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_dispatch_controls_learn_dispatch_style"
    ON "dispatch_controls"("organization_id", "business_unit_id")
    WHERE "learn_dispatch_style" IS TRUE;
