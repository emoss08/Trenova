-- An empty driver sent toward an area where the freight forecast expects more loads than
-- drivers, from the freight outlook. While a positioning is active and newer than the
-- driver's latest assignment, the forecast and the planner count the driver at the target
-- from arrive_at. A driver has at most one active positioning; a new one replaces it.
CREATE TABLE IF NOT EXISTS "driver_positionings"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "from_cell" bigint NOT NULL DEFAULT 0,
    "from_label" varchar(255) NOT NULL DEFAULT '',
    "to_cell" bigint NOT NULL,
    "to_label" varchar(255) NOT NULL DEFAULT '',
    "to_latitude" double precision NOT NULL,
    "to_longitude" double precision NOT NULL,
    "slot_start" bigint NOT NULL,
    "free_at" bigint NOT NULL,
    "arrive_at" bigint NOT NULL,
    "miles" double precision NOT NULL DEFAULT 0,
    "value_cents" bigint NOT NULL DEFAULT 0,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "created_by_id" varchar(100),
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_driver_positionings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_driver_positionings_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_driver_positionings_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_driver_positionings_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id")
        REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_driver_positionings_status" CHECK ("status" IN ('Active', 'Canceled', 'Replaced')),
    CONSTRAINT "ck_driver_positionings_arrival" CHECK ("arrive_at" >= "free_at")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_driver_positionings_active_worker"
    ON "driver_positionings"("organization_id", "business_unit_id", "worker_id")
    WHERE "status" = 'Active';

CREATE INDEX IF NOT EXISTS "idx_driver_positionings_active"
    ON "driver_positionings"("organization_id", "business_unit_id", "slot_start")
    WHERE "status" = 'Active';

COMMENT ON TABLE "driver_positionings" IS 'Empty drivers sent toward an area where freight is expected, which the forecast and the planner count at the target until they arrive, are reassigned or it is called off';

ALTER TABLE "demand_forecast_rates"
    ADD COLUMN IF NOT EXISTS "recent" jsonb NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN "demand_forecast_rates"."recent" IS 'Loads in this area and slot in each of the last eight weeks, oldest first; set on area rates only';

-- The forecast's track record: for each of the last fourteen days, the range of loads the
-- whole network was forecast to bring (from the weeks before that day's week, so out of
-- sample) and the loads that came. Replaced as a whole per tenant on every fit.
CREATE TABLE IF NOT EXISTS "demand_forecast_days"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "day_start" bigint NOT NULL,
    "low" integer NOT NULL,
    "high" integer NOT NULL,
    "mean" double precision NOT NULL,
    "actual" integer NOT NULL,
    "fitted_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_demand_forecast_days" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_demand_forecast_days_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_demand_forecast_days_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_demand_forecast_days_range" CHECK ("low" <= "high")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_demand_forecast_days_day"
    ON "demand_forecast_days"("organization_id", "business_unit_id", "day_start");

COMMENT ON TABLE "demand_forecast_days" IS 'Out-of-sample daily network forecast range against actual loads, for the freight outlook track record';
