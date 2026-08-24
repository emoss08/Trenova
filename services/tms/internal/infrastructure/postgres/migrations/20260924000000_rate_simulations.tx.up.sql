CREATE TYPE "rate_simulation_status_enum" AS ENUM(
    'Pending',
    'Running',
    'Completed',
    'Failed',
    'Canceled'
);

--bun:split
CREATE TABLE IF NOT EXISTS "rate_simulations"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "rate_agreement_id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(150) NOT NULL,
    "description" TEXT,
    "status" rate_simulation_status_enum NOT NULL DEFAULT 'Pending',
    "party_type" rate_agreement_party_type_enum NOT NULL DEFAULT 'Customer',
    "sample_from" BIGINT NOT NULL,
    "sample_to" BIGINT NOT NULL,
    "sample_limit" INTEGER NOT NULL DEFAULT 0,
    "summary" JSONB,
    "rule_coverage" JSONB,
    "error" TEXT,
    "started_at" BIGINT,
    "completed_at" BIGINT,
    "requested_by_id" VARCHAR(100),
    "workflow_id" VARCHAR(255),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_rate_simulations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_simulations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_simulations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- A window that contains nothing would report a zero delta, which reads as
    -- "this change costs nothing" rather than "this measured nothing".
    CONSTRAINT "chk_rate_simulations_window" CHECK ("sample_to" > "sample_from"),
    CONSTRAINT "chk_rate_simulations_sample_limit" CHECK ("sample_limit" >= 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_simulations_agreement" ON "rate_simulations"("organization_id", "business_unit_id", "rate_agreement_id", "created_at" DESC);

--bun:split
-- Finding the runs still in flight is what a worker and a status screen both
-- ask, and the terminal rows outnumber them permanently.
CREATE INDEX IF NOT EXISTS "idx_rate_simulations_active" ON "rate_simulations"("organization_id", "business_unit_id", "status")
WHERE
    "status" IN ('Pending', 'Running');

--bun:split
CREATE TABLE IF NOT EXISTS "rate_simulation_results"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "rate_simulation_id" VARCHAR(100) NOT NULL,
    "shipment_id" VARCHAR(100) NOT NULL,
    "pro_number" VARCHAR(100),
    "customer_id" VARCHAR(100),
    "lane_key" VARCHAR(160),
    "equipment_type_id" VARCHAR(100),
    "before_amount" NUMERIC(19, 4) NOT NULL DEFAULT 0,
    "after_amount" NUMERIC(19, 4) NOT NULL DEFAULT 0,
    "delta" NUMERIC(19, 4) NOT NULL DEFAULT 0,
    "delta_percent" NUMERIC(9, 4) NOT NULL DEFAULT 0,
    "outcome" rate_quote_outcome_enum NOT NULL,
    "before_rule_id" VARCHAR(100),
    "after_rule_id" VARCHAR(100),
    "error" TEXT,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_rate_simulation_results" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_simulation_results_simulation" FOREIGN KEY ("rate_simulation_id", "business_unit_id", "organization_id") REFERENCES "rate_simulations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- The grid reads one simulation's rows sorted by how far each shipment moved,
-- because the outliers are what somebody is looking for.
CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_by_move" ON "rate_simulation_results"("rate_simulation_id", "delta" DESC);

--bun:split
-- Grouping by customer and by lane is how a change is judged before it ships.
CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_customer" ON "rate_simulation_results"("rate_simulation_id", "customer_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_lane" ON "rate_simulation_results"("rate_simulation_id", "lane_key");
