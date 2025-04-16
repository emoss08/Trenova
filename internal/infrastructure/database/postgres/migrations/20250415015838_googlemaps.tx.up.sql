CREATE TABLE IF NOT EXISTS "google_maps_config"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- API KEY
    "api_key" text NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_google_maps_config" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_google_maps_config_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_google_maps_config_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one shipment control per organization
    CONSTRAINT "uq_google_maps_config_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_google_maps_config_business_unit" ON "google_maps_config"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_google_maps_config_created_at" ON "google_maps_config"("created_at", "updated_at");

-- Add comment to describe the table purpose
COMMENT ON TABLE google_maps_config IS 'Stores configuration for Google Maps API';

--bun:split
CREATE OR REPLACE FUNCTION google_maps_config_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS "google_maps_config_update_timestamp_trigger" ON "google_maps_config";

CREATE TRIGGER "google_maps_config_update_timestamp_trigger"
    BEFORE UPDATE ON "google_maps_config"
    FOR EACH ROW
    EXECUTE FUNCTION "google_maps_config_update_timestamp"();

--bun:split
ALTER TABLE "google_maps_config"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "google_maps_config"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;
