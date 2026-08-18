-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260924000000_rate_simulations.tx.up.sql

CREATE TABLE IF NOT EXISTS "rate_simulations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "party_type" TEXT NOT NULL DEFAULT 'Customer',
    "sample_from" INTEGER NOT NULL,
    "sample_to" INTEGER NOT NULL,
    "sample_limit" INTEGER NOT NULL DEFAULT 0,
    "summary" TEXT,
    "rule_coverage" TEXT,
    "error" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "requested_by_id" TEXT,
    "workflow_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_simulations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_simulations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_simulations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_simulations_window" CHECK ("sample_to" > "sample_from"),
    CONSTRAINT "chk_rate_simulations_sample_limit" CHECK ("sample_limit" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_simulations_agreement" ON "rate_simulations" ("organization_id", "business_unit_id", "rate_agreement_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_simulations_active" ON "rate_simulations" ("organization_id", "business_unit_id", "status")WHERE
    "status" IN ('Pending', 'Running');

--bun:split

CREATE TABLE IF NOT EXISTS "rate_simulation_results"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_simulation_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "pro_number" TEXT,
    "customer_id" TEXT,
    "lane_key" TEXT,
    "equipment_type_id" TEXT,
    "before_amount" REAL NOT NULL DEFAULT 0,
    "after_amount" REAL NOT NULL DEFAULT 0,
    "delta" REAL NOT NULL DEFAULT 0,
    "delta_percent" REAL NOT NULL DEFAULT 0,
    "outcome" TEXT NOT NULL,
    "before_rule_id" TEXT,
    "after_rule_id" TEXT,
    "error" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_simulation_results" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_simulation_results_simulation" FOREIGN KEY ("rate_simulation_id", "business_unit_id", "organization_id") REFERENCES "rate_simulations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_by_move" ON "rate_simulation_results" ("rate_simulation_id", "delta" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_customer" ON "rate_simulation_results" ("rate_simulation_id", "customer_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_simulation_results_lane" ON "rate_simulation_results" ("rate_simulation_id", "lane_key");
