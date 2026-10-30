-- A service failure stores its stop's location so a location's failures are listed and counted
-- newest first from an index on the failure, instead of joining every stop of the tenant to find
-- the ones at that location. A trigger sets it from the stop whenever a failure is written, and a
-- trigger on stops moves a stop's failures with it when its location changes, so every writer
-- (the evaluator, imports, seeds, fixtures) keeps it equal to the stop's; existing rows are derived
-- here.
ALTER TABLE "service_failures"
    ADD COLUMN IF NOT EXISTS "location_id" varchar(100);

UPDATE "service_failures" AS sf
SET "location_id" = st."location_id"
FROM "stops" AS st
WHERE st."id" = sf."stop_id"
    AND st."organization_id" = sf."organization_id"
    AND st."business_unit_id" = sf."business_unit_id";

ALTER TABLE "service_failures"
    ALTER COLUMN "location_id" SET NOT NULL;

CREATE OR REPLACE FUNCTION "service_failures_stop_location"()
    RETURNS TRIGGER
    AS $$
BEGIN
    SELECT st."location_id" INTO NEW."location_id"
    FROM "stops" AS st
    WHERE st."id" = NEW."stop_id"
        AND st."organization_id" = NEW."organization_id"
        AND st."business_unit_id" = NEW."business_unit_id";
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_service_failures_stop_location" ON "service_failures";

CREATE TRIGGER "trg_service_failures_stop_location"
    BEFORE INSERT OR UPDATE OF "stop_id" ON "service_failures"
    FOR EACH ROW
    EXECUTE FUNCTION "service_failures_stop_location"();

CREATE OR REPLACE FUNCTION "stops_service_failure_location"()
    RETURNS TRIGGER
    AS $$
BEGIN
    UPDATE "service_failures"
    SET "location_id" = NEW."location_id"
    WHERE "stop_id" = NEW."id"
        AND "organization_id" = NEW."organization_id"
        AND "business_unit_id" = NEW."business_unit_id";
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_stops_service_failure_location" ON "stops";

CREATE TRIGGER "trg_stops_service_failure_location"
    AFTER UPDATE OF "location_id" ON "stops"
    FOR EACH ROW
    WHEN (OLD."location_id" IS DISTINCT FROM NEW."location_id")
    EXECUTE FUNCTION "stops_service_failure_location"();
