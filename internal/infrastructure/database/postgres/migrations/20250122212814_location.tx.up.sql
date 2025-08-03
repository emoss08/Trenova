--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "locations" (
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
    "postal_code" us_postal_code,
    "longitude" float,
    "latitude" float,
    "place_id" text,
    "is_geocoded" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_locations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_locations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_locations_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_locations_location_category" FOREIGN KEY ("location_category_id", "business_unit_id", "organization_id") REFERENCES "location_categories" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for locations table
CREATE UNIQUE INDEX "idx_locations_code" ON "locations" (lower("code"), "organization_id");

CREATE INDEX "idx_locations_name" ON "locations" ("name");

CREATE INDEX "idx_locations_business_unit" ON "locations" ("business_unit_id");

CREATE INDEX "idx_locations_organization" ON "locations" ("organization_id");

CREATE INDEX "idx_locations_created_updated" ON "locations" ("created_at", "updated_at");

COMMENT ON TABLE "locations" IS 'Stores information about locations';

--bun:split
ALTER TABLE "stops"
    ADD COLUMN "location_id" varchar(100) NOT NULL;

ALTER TABLE "stops"
    ADD CONSTRAINT "fk_stops_location" FOREIGN KEY ("location_id", "business_unit_id", "organization_id") REFERENCES "locations" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "locations"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_search ON locations USING GIN (search_vector);

--bun:split
CREATE OR REPLACE FUNCTION locations_search_vector_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B') || setweight(to_tsvector('simple', COALESCE(NEW.address_line_1, '')), 'B') || setweight(to_tsvector('simple', COALESCE(NEW.address_line_2, '')), 'B') || setweight(to_tsvector('simple', COALESCE(NEW.city, '')), 'B') || setweight(to_tsvector('simple', COALESCE(NEW.postal_code::text, '')), 'C');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS locations_search_vector_trigger ON locations;

--bun:split
CREATE TRIGGER locations_search_vector_trigger
    BEFORE INSERT OR UPDATE ON locations
    FOR EACH ROW
    EXECUTE FUNCTION locations_search_vector_update ();

--bun:split
ALTER TABLE locations
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE locations
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE locations
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_code ON locations USING gin (code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_name ON locations USING gin (name gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_description ON locations USING gin (description gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_address_line_1 ON locations USING gin (address_line_1 gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_address_line_2 ON locations USING gin (address_line_2 gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_city ON locations USING gin (city gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_postal_code ON locations USING gin (postal_code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_locations_trgm_code_name_description ON locations USING gin ((code || ' ' || name || ' ' || description) gin_trgm_ops);

--bun:split
