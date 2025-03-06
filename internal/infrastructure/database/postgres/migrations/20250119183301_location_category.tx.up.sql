-- Enum for location category types with descriptions
CREATE TYPE "location_category_type" AS ENUM (
    'Terminal', -- A terminal is a facility where goods are received, stored, and distributed.
    'Warehouse', -- A warehouse is a facility where goods are stored and distributed.
    'DistributionCenter', -- A distribution center is a facility where goods are received, stored, and distributed.
    'TruckStop', -- A truck stop is a facility where trucks can stop to rest and refuel.
    'RestArea', -- A rest area is a facility where trucks can stop to rest and refuel.
    'CustomerLocation', -- A customer location is a facility where goods are received, stored, and distributed.
    'Port', -- A port is a facility where goods are received, stored, and distributed.
    'RailYard', -- A rail yard is a facility where goods are received, stored, and distributed.
    'MaintenanceFacility' -- A maintenance facility is a facility where goods are received, stored, and distributed.
);

--bun:split
-- Enum for facility types with descriptions
CREATE TYPE "facility_type" AS ENUM (
    'CrossDock', -- A cross dock is a facility where goods are received, stored, and distributed.
    'StorageWarehouse', -- A storage warehouse is a facility where goods are stored and distributed.
    'ColdStorage', -- A cold storage is a facility where goods are stored and distributed.
    'HazmatFacility', -- A hazmat facility is a facility where goods are stored and distributed.
    'IntermodalFacility' -- An intermodal facility is a facility where goods are received, stored, and distributed.
);

--bun:split
CREATE TABLE IF NOT EXISTS "location_categories" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "name" varchar(100) NOT NULL,
    "description" text,
    "type" location_category_type NOT NULL,
    "facility_type" facility_type,
    "has_secure_parking" boolean NOT NULL DEFAULT FALSE,
    "requires_appointment" boolean NOT NULL DEFAULT FALSE,
    "allows_overnight" boolean NOT NULL DEFAULT FALSE,
    "has_restroom" boolean NOT NULL DEFAULT FALSE,
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_location_categories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_location_categories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_location_categories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for location_categories table
CREATE UNIQUE INDEX "idx_location_categories_name" ON "location_categories" (lower("name"), "organization_id");

CREATE INDEX "idx_location_categories_business_unit" ON "location_categories" ("business_unit_id");

CREATE INDEX "idx_location_categories_organization" ON "location_categories" ("organization_id");

CREATE INDEX "idx_location_categories_created_updated" ON "location_categories" ("created_at", "updated_at");

COMMENT ON TABLE "location_categories" IS 'Stores information about location categories';

