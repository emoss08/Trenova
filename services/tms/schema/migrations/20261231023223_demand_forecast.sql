-- The organization's freight forecast, refitted nightly from its own shipment history.
-- Areas are web mercator tiles at zoom 8 and slots are four-hour blocks from Monday 00:00
-- UTC, the same keys as dispatch_market_values. A rate is the posterior of an area's (or a
-- customer's share of an area's) loads per week in one slot: Gamma(alpha, beta). Rows are
-- replaced as a whole per tenant on every fit, so they carry no version.
CREATE TABLE IF NOT EXISTS "demand_forecast_rates"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100),
    "cell" bigint NOT NULL,
    "slot" smallint NOT NULL,
    "alpha" double precision NOT NULL,
    "beta" double precision NOT NULL,
    "loads" integer NOT NULL DEFAULT 0,
    "fitted_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_demand_forecast_rates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_demand_forecast_rates_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_demand_forecast_rates_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_demand_forecast_rates_slot" CHECK ("slot" >= 0 AND "slot" < 42),
    CONSTRAINT "ck_demand_forecast_rates_posterior" CHECK ("alpha" > 0 AND "beta" > 0)
);

CREATE INDEX IF NOT EXISTS "idx_demand_forecast_rates_cell"
    ON "demand_forecast_rates"("organization_id", "business_unit_id", "cell", "slot");

COMMENT ON TABLE "demand_forecast_rates" IS 'Loads per week an area (or a customer in it) brings in each four-hour slot of the week, as a Gamma posterior fitted from the organization''s history';

-- What an area is called and what a load from it is usually worth.
CREATE TABLE IF NOT EXISTS "demand_forecast_areas"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "cell" bigint NOT NULL,
    "label" varchar(255) NOT NULL DEFAULT '',
    "median_revenue_cents" bigint NOT NULL DEFAULT 0,
    "loads" integer NOT NULL DEFAULT 0,
    "fitted_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_demand_forecast_areas" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_demand_forecast_areas_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_demand_forecast_areas_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_demand_forecast_areas_cell"
    ON "demand_forecast_areas"("organization_id", "business_unit_id", "cell");

COMMENT ON TABLE "demand_forecast_areas" IS 'Each forecast area''s name (its busiest origin city) and the median revenue of a load from it';

-- Share of a customer's loads booked at least N hours before pickup, on fixed lead buckets
-- (0, 2, 4, 8, 12, 24, 36, 48, 72, 96, 120, 168, 336 hours). A null customer is the
-- organization's own curve, which small customers are shrunk toward.
CREATE TABLE IF NOT EXISTS "demand_booking_curves"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100),
    "shares" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "loads" integer NOT NULL DEFAULT 0,
    "fitted_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_demand_booking_curves" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_demand_booking_curves_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_demand_booking_curves_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_demand_booking_curves_tenant"
    ON "demand_booking_curves"("organization_id", "business_unit_id");

COMMENT ON TABLE "demand_booking_curves" IS 'How far ahead each customer books its loads, used to tell freight still to come from freight already booked';

-- How the forecast did when tested on the organization's last weeks of history. Plans use
-- the forecast only when status is ready.
CREATE TABLE IF NOT EXISTS "demand_forecast_quality"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" varchar(30) NOT NULL,
    "history_weeks" integer NOT NULL DEFAULT 0,
    "history_loads" bigint NOT NULL DEFAULT 0,
    "weeks_tested" integer NOT NULL DEFAULT 0,
    "loads_tested" bigint NOT NULL DEFAULT 0,
    "interval_coverage" double precision NOT NULL DEFAULT 0,
    "mae" double precision NOT NULL DEFAULT 0,
    "naive_mae" double precision NOT NULL DEFAULT 0,
    "skill" double precision NOT NULL DEFAULT 0,
    "fitted_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_demand_forecast_quality" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_demand_forecast_quality_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_demand_forecast_quality_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_demand_forecast_quality_status" CHECK (
        "status" IN ('insufficientHistory', 'noSkill', 'ready')
    )
);

COMMENT ON TABLE "demand_forecast_quality" IS 'The freight forecast''s rolling-origin backtest on the organization''s own history, and whether plans may use it';
