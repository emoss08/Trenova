-- The standing dispatch plan's prices, one row per move it considered: the
-- driver it picked and what that pairing costs, what leaving the move would
-- cost, what that driver is worth to the plan, and how much more the plan
-- costs if each next-best driver takes it instead. Rewritten for the moves a
-- plan covers each time it runs, and read by the board in one batch.
CREATE TABLE IF NOT EXISTS "dispatch_plan_prices"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "move_id" varchar(100) NOT NULL,
    "worker_id" varchar(100),
    "worker_name" varchar(255) NOT NULL DEFAULT '',
    "cost_cents" bigint NOT NULL DEFAULT 0,
    "coverage_price_cents" bigint NOT NULL DEFAULT 0,
    "driver_value_cents" bigint NOT NULL DEFAULT 0,
    "uncovered_reason" varchar(50) NOT NULL DEFAULT '',
    "alternatives" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "generated_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_dispatch_plan_prices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_dispatch_plan_prices_move" FOREIGN KEY ("move_id", "organization_id", "business_unit_id")
        REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_dispatch_plan_prices_reason" CHECK ("uncovered_reason" IN (
        '', 'NoCandidate', 'Disqualified', 'BeyondDeadhead', 'LocationUnknown', 'NotWorthIt', 'Outbid'
    ))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_dispatch_plan_prices_move"
    ON "dispatch_plan_prices"("organization_id", "business_unit_id", "move_id");

CREATE INDEX IF NOT EXISTS "idx_dispatch_plan_prices_worker"
    ON "dispatch_plan_prices"("organization_id", "business_unit_id", "worker_id")
    WHERE "worker_id" IS NOT NULL;

COMMENT ON TABLE "dispatch_plan_prices" IS 'The standing dispatch plan''s prices per move: the planned driver and cost, the coverage price, the driver''s value to the plan and the exact extra cost of each alternative';
