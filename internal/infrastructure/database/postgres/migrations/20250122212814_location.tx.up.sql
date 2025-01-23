CREATE TABLE IF NOT EXISTS "locations"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "location_category_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(100) NOT NULL,
    "name" text,
    "description" text,
    "address_line_1" varchar(150),
    "address_line_2" varchar(150),
    "city" varchar(100),
    "postal_code" varchar(10),
    "longitude" float,
    "latitude" float,
    "place_id" varchar(100),
    "is_geocoded" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_locations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_locations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_locations_location_category" FOREIGN KEY ("location_category_id", "business_unit_id", "organization_id") REFERENCES "location_categories"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for locations table
CREATE UNIQUE INDEX "idx_locations_code" ON "locations"(lower("code"), "organization_id");

CREATE INDEX "idx_locations_name" ON "locations"("name");

CREATE INDEX "idx_locations_business_unit" ON "locations"("business_unit_id");

CREATE INDEX "idx_locations_organization" ON "locations"("organization_id");

CREATE INDEX "idx_locations_created_updated" ON "locations"("created_at", "updated_at");

COMMENT ON TABLE "locations" IS 'Stores information about locations';

