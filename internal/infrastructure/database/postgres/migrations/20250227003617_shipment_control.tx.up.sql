-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

-- Enums with documentation
CREATE TYPE "service_failure_enum" AS ENUM(
    'Pickup',
    'Delivery',
    'Both'
);

--bun:split
CREATE TYPE "auto_assignment_strategy_enum" AS ENUM(
    'Proximity',
    'Availability',
    'LoadBalancing'
);

--bun:split
CREATE TYPE "compliance_enforcement_level_enum" AS ENUM(
    'Warning',
    'Block',
    'Audit'
);

--bun:split
CREATE TABLE IF NOT EXISTS "shipment_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Auto Assignment Related Fields
    "enable_auto_assignment" boolean NOT NULL DEFAULT TRUE,
    "auto_assignment_strategy" "auto_assignment_strategy_enum" NOT NULL DEFAULT 'Proximity',
    -- Service Failure Related Fields
    "record_service_failures" boolean NOT NULL DEFAULT FALSE,
    "service_failure_grace_period" integer DEFAULT 30,
    -- Delay Shipment Related Fields
    "auto_delay_shipments" boolean NOT NULL DEFAULT TRUE,
    "auto_delay_shipments_threshold" integer DEFAULT 30,
    -- Compliance Controls
    "enforce_hos_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_driver_qualification_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_medical_cert_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_hazmat_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_drug_and_alcohol_compliance" boolean NOT NULL DEFAULT TRUE,
    "compliance_enforcement_level" "compliance_enforcement_level_enum" NOT NULL DEFAULT 'Warning',
    -- Detention Tracking
    "track_detention_time" boolean NOT NULL DEFAULT TRUE,
    "auto_generate_detention_charges" boolean NOT NULL DEFAULT TRUE,
    "detention_threshold" integer NOT NULL DEFAULT 30,
    -- Performance Metrics
    "on_time_delivery_target" float,
    "service_failure_target" float,
    "track_customer_rejections" boolean NOT NULL DEFAULT FALSE,
    -- Misc
    "check_for_duplicate_bols" boolean NOT NULL DEFAULT TRUE,
    "allow_move_removals" boolean NOT NULL DEFAULT TRUE,
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

--bun:split
ALTER TABLE shipment_controls
    ADD COLUMN IF NOT EXISTS check_hazmat_segregation BOOLEAN NOT NULL DEFAULT TRUE;

-- Add comment
COMMENT ON COLUMN shipment_controls.check_hazmat_segregation IS 'Controls whether hazardous material segregation validation is performed';

