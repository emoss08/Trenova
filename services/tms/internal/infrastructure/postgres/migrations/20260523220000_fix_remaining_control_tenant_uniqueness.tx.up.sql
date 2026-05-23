DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "data_entry_controls"
        GROUP BY "organization_id", "business_unit_id"
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'data_entry_controls contains duplicate rows for organization_id/business_unit_id; deduplicate before applying tenant uniqueness';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM "document_controls"
        GROUP BY "organization_id", "business_unit_id"
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'document_controls contains duplicate rows for organization_id/business_unit_id; deduplicate before applying tenant uniqueness';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM "shipment_controls"
        GROUP BY "organization_id", "business_unit_id"
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'shipment_controls contains duplicate rows for organization_id/business_unit_id; deduplicate before applying tenant uniqueness';
    END IF;
END $$;

ALTER TABLE "data_entry_controls"
    DROP CONSTRAINT IF EXISTS "uq_data_entry_controls_organization";

ALTER TABLE "data_entry_controls"
    ADD CONSTRAINT "uq_data_entry_controls_tenant"
    UNIQUE ("organization_id", "business_unit_id");

ALTER TABLE "document_controls"
    DROP CONSTRAINT IF EXISTS "uq_document_controls_organization";

ALTER TABLE "document_controls"
    ADD CONSTRAINT "uq_document_controls_tenant"
    UNIQUE ("organization_id", "business_unit_id");

ALTER TABLE "shipment_controls"
    DROP CONSTRAINT IF EXISTS "uq_shipment_controls_organization";

ALTER TABLE "shipment_controls"
    ADD CONSTRAINT "uq_shipment_controls_tenant"
    UNIQUE ("organization_id", "business_unit_id");
