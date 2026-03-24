DROP INDEX IF EXISTS "uq_assignments_move_tenant_active";

ALTER TABLE "assignments"
    DROP COLUMN IF EXISTS "archived_at";

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assignments_move_tenant"
    ON "assignments"("shipment_move_id", "organization_id", "business_unit_id");
