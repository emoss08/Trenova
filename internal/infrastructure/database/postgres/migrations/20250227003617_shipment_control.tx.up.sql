CREATE TABLE IF NOT EXISTS "shipment_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Service Failure Related Fields
    "record_service_failures" boolean NOT NULL DEFAULT TRUE,
    "service_failure_grace_period" bigint NOT NULL DEFAULT 10,
    -- Delay Shipment Related Fields
    "auto_delay_shipments" boolean NOT NULL DEFAULT TRUE,
    "auto_delay_shipments_threshold" bigint NOT NULL DEFAULT 10,
    -- Compliance Controls
    "enforce_hos_compliance" boolean NOT NULL DEFAULT TRUE,
    -- Detention Tracking
    "track_detention_time" boolean NOT NULL DEFAULT TRUE,
    "detention_threshold" bigint NOT NULL DEFAULT 10,
    -- Misc
    "check_for_duplicate_bols" boolean NOT NULL DEFAULT TRUE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipment_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one shipment control per organization
    CONSTRAINT "uq_shipment_controls_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_controls_business_unit" ON "shipment_controls"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_shipment_controls_created_at" ON "shipment_controls"("created_at", "updated_at");

-- Add comment to describe the table purpose
COMMENT ON TABLE shipment_controls IS 'Stores configuration for shipment controls and validation rules';

--bun:split
CREATE OR REPLACE FUNCTION shipment_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS shipment_controls_update_timestamp_trigger ON shipment_controls;

CREATE TRIGGER shipment_controls_update_timestamp_trigger
    BEFORE UPDATE ON shipment_controls
    FOR EACH ROW
    EXECUTE FUNCTION shipment_controls_update_timestamp();

--bun:split
ALTER TABLE shipment_controls
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE shipment_controls
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

