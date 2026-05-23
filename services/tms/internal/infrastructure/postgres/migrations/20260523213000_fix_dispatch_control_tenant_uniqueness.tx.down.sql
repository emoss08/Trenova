ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "uq_dispatch_controls_tenant";

ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "uq_dispatch_controls_organization"
    UNIQUE ("organization_id");
