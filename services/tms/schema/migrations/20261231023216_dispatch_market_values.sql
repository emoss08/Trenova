-- What a driver free in a market cell at a time of the week has been worth to the
-- organization's plans: each plan's driver dual prices, averaged per week and folded
-- across weeks as an exponential moving average. Cells are web mercator tiles at zoom
-- 8 (about 120 km across in the US), slots are four-hour blocks from Monday 00:00 UTC,
-- so a tenant holds at most 42 rows per cell it has seen drivers in. The planner reads
-- the rows of the destination cells of the moves it prices.
CREATE TABLE IF NOT EXISTS "dispatch_market_values"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "cell" bigint NOT NULL,
    "slot" smallint NOT NULL,
    "value_cents" double precision NOT NULL DEFAULT 0,
    "weeks" integer NOT NULL DEFAULT 0,
    "week_index" bigint NOT NULL DEFAULT 0,
    "week_sum_cents" double precision NOT NULL DEFAULT 0,
    "week_samples" integer NOT NULL DEFAULT 0,
    "observations" bigint NOT NULL DEFAULT 0,
    "observed_at" bigint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_dispatch_market_values" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dispatch_market_values_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dispatch_market_values_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_dispatch_market_values_slot" CHECK ("slot" >= 0 AND "slot" < 42),
    CONSTRAINT "ck_dispatch_market_values_counts" CHECK (
        "weeks" >= 0 AND "week_samples" >= 0 AND "observations" >= 0
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dispatch_market_values_cell_slot"
    ON "dispatch_market_values"("organization_id", "business_unit_id", "cell", "slot");

COMMENT ON TABLE "dispatch_market_values" IS 'What a driver free in a market cell at a time of the week has been worth to the plan, learned from the plan''s own driver dual prices';
