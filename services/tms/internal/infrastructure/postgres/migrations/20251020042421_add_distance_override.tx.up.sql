CREATE TABLE IF NOT EXISTS "distance_overrides"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "origin_location_id" varchar(100) NOT NULL,
    "destination_location_id" varchar(100) NOT NULL,
    "customer_id" varchar(100),
    "distance" float NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_distance_overrides" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_overrides_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_distance_overrides_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_distance_overrides_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- General business unit and organization index
CREATE INDEX IF NOT EXISTS "idx_distance_overrides_business_unit" ON "distance_overrides"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_distance_overrides_created_at" ON "distance_overrides"("created_at", "updated_at");

--bun:split
COMMENT ON TABLE distance_overrides IS 'Stores distance overrides for shipments';

--bun:split
CREATE OR REPLACE FUNCTION distance_overrides_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS distance_overrides_update_trigger ON distance_overrides;

--bun:split
CREATE TRIGGER distance_overrides_update_trigger
    BEFORE UPDATE ON distance_overrides
    FOR EACH ROW
    EXECUTE FUNCTION distance_overrides_update_timestamps();

--bun:split
ALTER TABLE distance_overrides
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE distance_overrides
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE distance_overrides
    ALTER COLUMN customer_id SET STATISTICS 1000;

ALTER TABLE "distance_overrides"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_distance_overrides_search_vector ON "distance_overrides" USING GIN(search_vector);

