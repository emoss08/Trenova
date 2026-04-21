CREATE TYPE "location_geofence_type_enum" AS ENUM (
    'auto',
    'circle',
    'rectangle',
    'draw'
);

--bun:split
ALTER TABLE "locations"
    ADD COLUMN "geofence_type" location_geofence_type_enum NOT NULL DEFAULT 'auto',
    ADD COLUMN "geofence_radius_meters" double precision,
    ADD COLUMN "geofence_geometry" geometry(Polygon, 4326);

--bun:split
UPDATE "locations"
SET
    "geofence_radius_meters" = 250,
    "geofence_geometry" = CASE
        WHEN "longitude" IS NOT NULL AND "latitude" IS NOT NULL THEN
            ST_Buffer(ST_SetSRID(ST_MakePoint("longitude", "latitude"), 4326)::geography, 250)::geometry
        ELSE NULL
    END
WHERE "geofence_radius_meters" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_locations_geofence_geometry" ON "locations" USING GIST("geofence_geometry");

--bun:split
COMMENT ON COLUMN "locations"."geofence_type" IS 'How the location geofence is defined: auto, circle, rectangle, or draw';

COMMENT ON COLUMN "locations"."geofence_radius_meters" IS 'Circle radius in meters for auto and circle geofences';

COMMENT ON COLUMN "locations"."geofence_geometry" IS 'Persisted polygon geometry representing the location geofence';
