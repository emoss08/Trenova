DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "dispatch_controls"
        GROUP BY "organization_id", "business_unit_id"
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'dispatch_controls contains duplicate rows for organization_id/business_unit_id; deduplicate before applying tenant uniqueness';
    END IF;
END $$;

ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "uq_dispatch_controls_organization";

ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "uq_dispatch_controls_tenant"
    UNIQUE ("organization_id", "business_unit_id");
