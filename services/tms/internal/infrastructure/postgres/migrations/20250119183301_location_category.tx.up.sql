CREATE TYPE "location_category_type" AS ENUM(
    'Terminal',
    'Warehouse',
    'DistributionCenter',
    'TruckStop',
    'RestArea',
    'CustomerLocation',
    'Port',
    'RailYard',
    'MaintenanceFacility'
);

--bun:split
CREATE TYPE "facility_type" AS ENUM(
    'CrossDock',
    'StorageWarehouse',
    'ColdStorage',
    'HazmatFacility',
    'IntermodalFacility'
);

--bun:split
CREATE TABLE IF NOT EXISTS "location_categories"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "type" location_category_type NOT NULL,
    "facility_type" facility_type,
    "has_secure_parking" boolean NOT NULL DEFAULT FALSE,
    "requires_appointment" boolean NOT NULL DEFAULT FALSE,
    "allows_overnight" boolean NOT NULL DEFAULT FALSE,
    "has_restroom" boolean NOT NULL DEFAULT FALSE,
    "color" varchar(10),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_location_categories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_location_categories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_location_categories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "idx_location_categories_name" ON "location_categories"(lower("name"), "organization_id");

CREATE INDEX "idx_location_categories_business_unit" ON "location_categories"("business_unit_id");

CREATE INDEX "idx_location_categories_organization" ON "location_categories"("organization_id");

CREATE INDEX "idx_location_categories_created_updated" ON "location_categories"("created_at", "updated_at");

COMMENT ON TABLE "location_categories" IS 'Stores information about location categories';

--bun:split
ALTER TABLE "location_categories"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_location_categories_search_vector ON "location_categories" USING GIN(search_vector);
