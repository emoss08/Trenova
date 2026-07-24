DROP TABLE IF EXISTS "telematics_feed_states";

DROP TABLE IF EXISTS "telematics_events";

DROP TABLE IF EXISTS "worker_hos_violations";

DROP TABLE IF EXISTS "worker_hos_states";

DROP TABLE IF EXISTS "telematics_vehicle_positions";

DROP INDEX IF EXISTS idx_tractors_external_id_tenant;

ALTER TABLE "tractors"
    DROP COLUMN IF EXISTS "external_id";
