-- The forecast now keeps its hours and weeks in the organization's own time zone, corrects its
-- level from the last two weeks and treats the organization's holidays as their own kind of day.
-- The fit records what it used, so every read forecasts on the same clock and calendar it was
-- fitted on.
ALTER TABLE "demand_forecast_quality"
    ADD COLUMN IF NOT EXISTS "timezone" varchar(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "level" double precision NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "holiday_factor" double precision NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "holidays" jsonb NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN "demand_forecast_quality"."timezone" IS 'The time zone the forecast''s four-hour slots and weeks were counted in at the last fit; empty is UTC';

COMMENT ON COLUMN "demand_forecast_quality"."level" IS 'How the last two weeks ran against the fitted rhythm, shrunk toward 1; the stored rates already carry it';

COMMENT ON COLUMN "demand_forecast_quality"."holiday_factor" IS 'The share of a normal day''s loads that arrive on one of the organization''s holidays, learned from the holidays in its history';

COMMENT ON COLUMN "demand_forecast_quality"."holidays" IS 'The organization''s holidays from the day before the fit to 14 days after, as day numbers in its own time zone';
