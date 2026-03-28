CREATE TABLE IF NOT EXISTS "document_controls" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "enable_document_intelligence" BOOLEAN NOT NULL DEFAULT TRUE,
    "enable_ocr" BOOLEAN NOT NULL DEFAULT TRUE,
    "enable_auto_classification" BOOLEAN NOT NULL DEFAULT TRUE,
    "enable_auto_document_type_associate" BOOLEAN NOT NULL DEFAULT TRUE,
    "enable_auto_create_document_types" BOOLEAN NOT NULL DEFAULT TRUE,
    "enable_shipment_draft_extraction" BOOLEAN NOT NULL DEFAULT TRUE,
    "shipment_draft_allowed_resources" VARCHAR(100)[] NOT NULL DEFAULT '{"shipment"}',
    "enable_full_text_indexing" BOOLEAN NOT NULL DEFAULT TRUE,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_document_controls_organization" UNIQUE ("organization_id")
);

CREATE INDEX IF NOT EXISTS "idx_document_controls_tenant" ON "document_controls"("business_unit_id", "organization_id");
CREATE INDEX IF NOT EXISTS "idx_document_controls_created_at" ON "document_controls"("created_at");
CREATE INDEX IF NOT EXISTS "idx_document_controls_updated_at" ON "document_controls"("updated_at");

COMMENT ON TABLE document_controls IS 'Stores per-organization settings for OCR, classification, document type association, shipment-draft extraction, and full-text indexing.';

--bun:split
CREATE OR REPLACE FUNCTION document_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS document_controls_update_timestamp_trigger ON document_controls;

CREATE TRIGGER document_controls_update_timestamp_trigger
    BEFORE UPDATE ON document_controls
    FOR EACH ROW
    EXECUTE FUNCTION document_controls_update_timestamp();
