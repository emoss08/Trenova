ALTER TABLE "shipment_controls"
    DROP CONSTRAINT IF EXISTS "uq_shipment_controls_tenant";

ALTER TABLE "shipment_controls"
    ADD CONSTRAINT "uq_shipment_controls_organization"
    UNIQUE ("organization_id");

ALTER TABLE "document_controls"
    DROP CONSTRAINT IF EXISTS "uq_document_controls_tenant";

ALTER TABLE "document_controls"
    ADD CONSTRAINT "uq_document_controls_organization"
    UNIQUE ("organization_id");

ALTER TABLE "data_entry_controls"
    DROP CONSTRAINT IF EXISTS "uq_data_entry_controls_tenant";

ALTER TABLE "data_entry_controls"
    ADD CONSTRAINT "uq_data_entry_controls_organization"
    UNIQUE ("organization_id");
