DROP INDEX IF EXISTS "idx_locations_geofence_geometry";

--bun:split
ALTER TABLE "locations"
    DROP COLUMN IF EXISTS "geofence_geometry",
    DROP COLUMN IF EXISTS "geofence_radius_meters",
    DROP COLUMN IF EXISTS "geofence_type";

--bun:split
DROP TYPE IF EXISTS "location_geofence_type_enum";
