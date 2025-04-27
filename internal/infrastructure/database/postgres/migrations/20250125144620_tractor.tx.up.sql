CREATE TYPE equipment_status_enum AS ENUM(
    'Available',
    'OutOfService',
    'AtMaintenance',
    'Sold'
);

--bun:split
CREATE TABLE IF NOT EXISTS "tractors"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Relationship identifiers (Non-Primary-Keys)
    "primary_worker_id" varchar(100) NOT NULL,
    "equipment_type_id" varchar(100) NOT NULL,
    "equipment_manufacturer_id" varchar(100) NOT NULL,
    "state_id" varchar(100),
    "fleet_code_id" varchar(100),
    "secondary_worker_id" varchar(100),
    -- Core fields
    "status" equipment_status_enum NOT NULL DEFAULT 'Available',
    "code" varchar(50) NOT NULL,
    "model" varchar(50),
    "make" varchar(50),
    "year" int,
    "license_plate_number" varchar(50),
    "registration_number" varchar(50),
    "registration_expiry" bigint,
    "vin" vin_code_optional,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_tractors" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_tractors_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_equipment_type" FOREIGN KEY ("equipment_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id", "business_unit_id", "organization_id") REFERENCES "equipment_manufacturers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_tractors_fleet_code" FOREIGN KEY ("fleet_code_id", "business_unit_id", "organization_id") REFERENCES "fleet_codes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_primary_worker" FOREIGN KEY ("primary_worker_id", "business_unit_id", "organization_id") REFERENCES "workers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_tractors_secondary_worker" FOREIGN KEY ("secondary_worker_id", "business_unit_id", "organization_id") REFERENCES "workers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Create unique index for code and organization_id
CREATE UNIQUE INDEX "idx_tractors_code" ON "tractors"(lower("code"), "organization_id");

-- Create unique index for primary_worker_id
CREATE UNIQUE INDEX IF NOT EXISTS "idx_tractors_primary_worker_id" ON "tractors"("primary_worker_id")
WHERE
    "primary_worker_id" IS NOT NULL;

-- Create index for status
CREATE INDEX "idx_tractors_status" ON "tractors"("status");

CREATE INDEX "idx_tractors_business_unit_organization" ON "tractors"("business_unit_id", "organization_id");

CREATE INDEX "idx_tractors_created_updated" ON "tractors"("created_at", "updated_at");

COMMENT ON TABLE "tractors" IS 'Stores information about tractors';

