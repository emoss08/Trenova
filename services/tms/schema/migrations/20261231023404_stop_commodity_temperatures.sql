-- A multi-temperature load keeps each drop's and each line's range: a stop says the range the
-- freight is kept at from that stop on, a commodity line the range its own freight needs. Both
-- are whole degrees Fahrenheit, as the shipment's range is, and the shipment's range is derived
-- from them when any is set.
ALTER TABLE "stops"
    ADD COLUMN IF NOT EXISTS "temperature_min" temperature_fahrenheit,
    ADD COLUMN IF NOT EXISTS "temperature_max" temperature_fahrenheit;

ALTER TABLE "stops"
    DROP CONSTRAINT IF EXISTS "chk_stops_temperature_range";

ALTER TABLE "stops"
    ADD CONSTRAINT "chk_stops_temperature_range"
    CHECK ("temperature_min" IS NULL OR "temperature_max" IS NULL OR "temperature_min" <= "temperature_max");

COMMENT ON COLUMN "stops"."temperature_min" IS 'Lowest temperature, in whole degrees Fahrenheit, the freight is kept at from this stop on';

COMMENT ON COLUMN "stops"."temperature_max" IS 'Highest temperature, in whole degrees Fahrenheit, the freight is kept at from this stop on';

ALTER TABLE "shipment_commodities"
    ADD COLUMN IF NOT EXISTS "temperature_min" temperature_fahrenheit,
    ADD COLUMN IF NOT EXISTS "temperature_max" temperature_fahrenheit;

ALTER TABLE "shipment_commodities"
    DROP CONSTRAINT IF EXISTS "chk_shipment_commodities_temperature_range";

ALTER TABLE "shipment_commodities"
    ADD CONSTRAINT "chk_shipment_commodities_temperature_range"
    CHECK ("temperature_min" IS NULL OR "temperature_max" IS NULL OR "temperature_min" <= "temperature_max");

COMMENT ON COLUMN "shipment_commodities"."temperature_min" IS 'Lowest temperature, in whole degrees Fahrenheit, this line''s freight must be kept at';

COMMENT ON COLUMN "shipment_commodities"."temperature_max" IS 'Highest temperature, in whole degrees Fahrenheit, this line''s freight must be kept at';
