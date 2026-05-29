DROP TABLE IF EXISTS "provider_location_resolutions";
DROP TABLE IF EXISTS "distance_calculation_runs";
DROP TABLE IF EXISTS "distance_profiles";

DROP INDEX IF EXISTS "idx_shipment_moves_distance_route_signature";

ALTER TABLE "shipment_moves"
    DROP COLUMN IF EXISTS "distance_metadata",
    DROP COLUMN IF EXISTS "distance_units",
    DROP COLUMN IF EXISTS "distance_routing_type",
    DROP COLUMN IF EXISTS "distance_data_version",
    DROP COLUMN IF EXISTS "distance_route_signature",
    DROP COLUMN IF EXISTS "distance_calculated_at",
    DROP COLUMN IF EXISTS "distance_provider",
    DROP COLUMN IF EXISTS "distance_source";

-- PostgreSQL cannot remove enum values safely without rebuilding the enum type.
