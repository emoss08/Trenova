CREATE TYPE "service_incident_type_enum" AS ENUM(
    'Never',
    'Pickup',
    'Delivery',
    'PickupDelivery',
    'AllExceptShipper'
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
CREATE TABLE IF NOT EXISTS "dispatch_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "enable_auto_assignment" boolean NOT NULL DEFAULT TRUE,
    "auto_assignment_strategy" "auto_assignment_strategy_enum" NOT NULL DEFAULT 'Proximity',
    "enforce_worker_assign" boolean NOT NULL DEFAULT TRUE,
    "enforce_trailer_continuity" boolean NOT NULL DEFAULT TRUE,
    "enforce_hos_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_worker_pta_restrictions" boolean NOT NULL DEFAULT TRUE,
    "enforce_worker_tractor_fleet_continuity" boolean NOT NULL DEFAULT TRUE,
    "enforce_driver_qualification_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_medical_cert_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_hazmat_compliance" boolean NOT NULL DEFAULT TRUE,
    "enforce_drug_and_alcohol_compliance" boolean NOT NULL DEFAULT TRUE,
    "compliance_enforcement_level" "compliance_enforcement_level_enum" NOT NULL DEFAULT 'Warning',
    "record_service_failures" "service_incident_type_enum" NOT NULL DEFAULT 'Never',
    "service_failure_target" float,
    "service_failure_grace_period" integer DEFAULT 30,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_dispatch_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dispatch_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dispatch_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_dispatch_controls_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_dispatch_controls_business_unit" ON "dispatch_controls"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_dispatch_controls_created_at" ON "dispatch_controls"("created_at", "updated_at");

COMMENT ON TABLE dispatch_controls IS 'Stores configuration for dispatch controls and validation rules';

--bun:split
CREATE OR REPLACE FUNCTION dispatch_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS dispatch_controls_update_timestamp_trigger ON dispatch_controls;

CREATE TRIGGER dispatch_controls_update_timestamp_trigger
    BEFORE UPDATE ON dispatch_controls
    FOR EACH ROW
    EXECUTE FUNCTION dispatch_controls_update_timestamp();

--bun:split
ALTER TABLE dispatch_controls
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE dispatch_controls
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

