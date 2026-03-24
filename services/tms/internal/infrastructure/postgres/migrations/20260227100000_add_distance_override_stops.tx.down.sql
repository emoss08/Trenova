DROP TABLE IF EXISTS "distance_override_stops";

DROP INDEX IF EXISTS "uq_distance_overrides_route_signature";

ALTER TABLE "distance_overrides"
    DROP COLUMN IF EXISTS "route_signature";
