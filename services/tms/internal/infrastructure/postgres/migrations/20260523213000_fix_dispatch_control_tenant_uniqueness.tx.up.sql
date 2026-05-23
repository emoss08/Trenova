ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "uq_dispatch_controls_organization";

ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "uq_dispatch_controls_tenant"
    UNIQUE ("organization_id", "business_unit_id");
