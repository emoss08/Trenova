DROP TABLE IF EXISTS "vehicle_inspections";

DROP INDEX IF EXISTS idx_trailers_external_id_tenant;

ALTER TABLE "trailers"
    DROP COLUMN IF EXISTS "external_id";
