ALTER TABLE "assignments"
    ADD COLUMN IF NOT EXISTS "archived_at" BIGINT;

DROP INDEX IF EXISTS "uq_assignments_move_tenant";

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assignments_move_tenant_active"
    ON "assignments"("shipment_move_id", "organization_id", "business_unit_id")
    WHERE "archived_at" IS NULL;
