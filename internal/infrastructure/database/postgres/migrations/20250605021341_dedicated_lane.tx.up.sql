--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "dedicated_lanes"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "service_type_id" varchar(100) NOT NULL,
    "shipment_type_id" varchar(100) NOT NULL,
    "trailer_type_id" varchar(100),
    "tractor_type_id" varchar(100),
    "origin_location_id" varchar(100) NOT NULL,
    "destination_location_id" varchar(100) NOT NULL,
    "primary_worker_id" varchar(100),
    "secondary_worker_id" varchar(100),
    -- Lane Configuration
    "auto_assign" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_dedicated_lanes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dedicated_lanes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lanes_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_trailer_type" FOREIGN KEY ("trailer_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_tractor_type" FOREIGN KEY ("tractor_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_primary_worker" FOREIGN KEY ("primary_worker_id", "business_unit_id", "organization_id") REFERENCES "workers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lanes_secondary_worker" FOREIGN KEY ("secondary_worker_id", "business_unit_id", "organization_id") REFERENCES "workers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_dedicated_lanes_name" ON "dedicated_lanes"(lower("name"), "organization_id");

--bun:split
-- Index for fast lookups during auto-assignment
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_auto_assign_lookup" ON "dedicated_lanes"("customer_id", "business_unit_id", "organization_id", "auto_assign")
WHERE
    "auto_assign" = TRUE;

--bun:split
-- Index for origin/destination lookups
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_locations" ON "dedicated_lanes"("origin_location_id", "destination_location_id");

--bun:split
-- Index for equipment type filtering
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_equipment" ON "dedicated_lanes"("tractor_type_id", "trailer_type_id");

--bun:split
-- Index for worker assignments
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_workers" ON "dedicated_lanes"("primary_worker_id", "secondary_worker_id");

--bun:split
-- General business unit and organization index
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_business_unit" ON "dedicated_lanes"("business_unit_id", "organization_id");

--bun:split
-- Index for tracking creation and updates
CREATE INDEX IF NOT EXISTS "idx_dedicated_lanes_created_at" ON "dedicated_lanes"("created_at", "updated_at");

--bun:split
COMMENT ON TABLE dedicated_lanes IS 'Stores dedicated lane configurations for automatic driver and equipment assignment';

COMMENT ON COLUMN dedicated_lanes.customer_id IS 'Customer this dedicated lane is configured for';

COMMENT ON COLUMN dedicated_lanes.origin_location_id IS 'Starting location for this dedicated lane';

COMMENT ON COLUMN dedicated_lanes.destination_location_id IS 'Ending location for this dedicated lane';

COMMENT ON COLUMN dedicated_lanes.primary_worker_id IS 'Primary driver automatically assigned to shipments on this lane';

COMMENT ON COLUMN dedicated_lanes.secondary_worker_id IS 'Optional secondary driver for team driving scenarios';

COMMENT ON COLUMN dedicated_lanes.auto_assign IS 'Whether to automatically assign workers and equipment when shipments match this lane';

--bun:split
-- Trigger function to auto-update timestamps
CREATE OR REPLACE FUNCTION dedicated_lanes_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS dedicated_lanes_update_trigger ON dedicated_lanes;

--bun:split
CREATE TRIGGER dedicated_lanes_update_trigger
    BEFORE UPDATE ON dedicated_lanes
    FOR EACH ROW
    EXECUTE FUNCTION dedicated_lanes_update_timestamps();

--bun:split
-- Ensure primary worker and secondary worker are different
ALTER TABLE dedicated_lanes
    ADD CONSTRAINT chk_dedicated_lanes_different_workers CHECK (primary_worker_id != secondary_worker_id);

--bun:split
-- Performance optimization: Set statistics for frequently queried columns
ALTER TABLE dedicated_lanes
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE dedicated_lanes
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE dedicated_lanes
    ALTER COLUMN customer_id SET STATISTICS 1000;

ALTER TABLE dedicated_lanes
    ALTER COLUMN auto_assign SET STATISTICS 1000;

--bun:split
-- Composite index for the most common query pattern (auto-assignment lookup)
CREATE INDEX idx_dedicated_lanes_assignment_lookup ON dedicated_lanes(organization_id, business_unit_id, customer_id, origin_location_id, destination_location_id)
WHERE
    auto_assign = TRUE;

