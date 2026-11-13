-- How much more a day's load total varies than the forecast's own range allows, measured on the
-- weeks the forecast was tested on. Ranges over the whole network are widened by it, so a
-- range that says 80% holds about 80% of the time.
ALTER TABLE "demand_forecast_quality"
    ADD COLUMN IF NOT EXISTS "dispersion" double precision NOT NULL DEFAULT 1;

COMMENT ON COLUMN "demand_forecast_quality"."dispersion" IS 'Variance of the tested days'' load totals over the variance the forecast expected, at least 1. Network-wide ranges and the track record are widened by it';
