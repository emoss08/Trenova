-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006000_agent_quality_sweep.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_suite_runs" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "trigger" TEXT NOT NULL DEFAULT 'Scheduled',
    "sweep_key" TEXT,
    "fingerprint" TEXT,
    "fingerprint_hash" TEXT NOT NULL,
    "fingerprint_changes" TEXT NOT NULL DEFAULT '[]',
    "suite_revision" TEXT NOT NULL,
    "sample_seed" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Running',
    "cases_total" INTEGER NOT NULL DEFAULT 0,
    "cases_passed" INTEGER NOT NULL DEFAULT 0,
    "cases_failed" INTEGER NOT NULL DEFAULT 0,
    "cases_skipped" INTEGER NOT NULL DEFAULT 0,
    "hard_failures" INTEGER NOT NULL DEFAULT 0,
    "deterministic_score" REAL,
    "judge_score" REAL,
    "quality_score" REAL,
    "baseline_score" REAL,
    "baseline_run_id" TEXT,
    "regression" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" REAL NOT NULL DEFAULT 0,
    "started_at" INTEGER NOT NULL,
    "finished_at" INTEGER,
    "comments" TEXT,
    "workflow_id" TEXT,
    "requested_by_user_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_suite_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_suite_runs_status" CHECK ("status" IN ('Running', 'Completed', 'Skipped', 'BudgetStopped', 'Failed')),
    CONSTRAINT "ck_agent_suite_runs_trigger" CHECK ("trigger" IN ('Scheduled', 'Manual')),
    CONSTRAINT "ck_agent_suite_runs_counts" CHECK ("cases_total" >= 0 AND "cases_passed" >= 0 AND "cases_failed" >= 0 AND "cases_skipped" >= 0 AND "hard_failures" >= 0 AND "cases_passed" + "cases_failed" + "cases_skipped" <= "cases_total"),
    CONSTRAINT "ck_agent_suite_runs_scores" CHECK (("deterministic_score" IS NULL OR "deterministic_score" BETWEEN 0 AND 1) AND ("judge_score" IS NULL OR "judge_score" BETWEEN 0 AND 1) AND ("quality_score" IS NULL OR "quality_score" BETWEEN 0 AND 1) AND ("baseline_score" IS NULL OR "baseline_score" BETWEEN 0 AND 1)),
    CONSTRAINT "ck_agent_suite_runs_cost" CHECK ("cost_usd" >= 0),
    CONSTRAINT "fk_agent_suite_runs_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suite_runs_requested_by" FOREIGN KEY ("requested_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_suite_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_suite_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_suite_runs_sweep_key"
    ON "agent_suite_runs" ("organization_id", "business_unit_id", "sweep_key")WHERE "sweep_key" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_suite_runs_agent"
    ON "agent_suite_runs" ("organization_id", "business_unit_id", "agent_definition_id", "started_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_suite_runs_started"
    ON "agent_suite_runs" ("organization_id", "business_unit_id", "started_at" DESC, "id" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_suite_runs_one_running"
    ON "agent_suite_runs" ("organization_id", "business_unit_id", "agent_definition_id")WHERE "status" = 'Running';

--bun:split

CREATE TABLE IF NOT EXISTS "agent_quality_controls" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "run_hour_local" INTEGER NOT NULL DEFAULT 2,
    "timezone" TEXT,
    "max_cases_per_agent" INTEGER NOT NULL DEFAULT 50,
    "nightly_budget_usd" REAL NOT NULL DEFAULT 5.00,
    "monthly_budget_usd" REAL NOT NULL DEFAULT 50.00,
    "judge_enabled" INTEGER NOT NULL DEFAULT 0,
    "judge_sample_rate" REAL NOT NULL DEFAULT 0.2,
    "regression_threshold" REAL NOT NULL DEFAULT 0.10,
    "min_cases" INTEGER NOT NULL DEFAULT 10,
    "force_rerun_days" INTEGER NOT NULL DEFAULT 7,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_quality_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_quality_controls_hour" CHECK ("run_hour_local" BETWEEN 0 AND 23),
    CONSTRAINT "ck_agent_quality_controls_cases" CHECK ("max_cases_per_agent" BETWEEN 1 AND 500 AND "min_cases" BETWEEN 1 AND 500),
    CONSTRAINT "ck_agent_quality_controls_budgets" CHECK ("nightly_budget_usd" >= 0 AND "monthly_budget_usd" >= "nightly_budget_usd"),
    CONSTRAINT "ck_agent_quality_controls_rates" CHECK ("judge_sample_rate" BETWEEN 0 AND 1 AND "regression_threshold" > 0 AND "regression_threshold" <= 1),
    CONSTRAINT "ck_agent_quality_controls_rerun" CHECK ("force_rerun_days" BETWEEN 1 AND 90),
    CONSTRAINT "fk_agent_quality_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_quality_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_quality_controls_tenant"
    ON "agent_quality_controls" ("organization_id", "business_unit_id");

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "suite_run_id" TEXT;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "suite_ordinal" INTEGER;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_evaluations_suite_ordinal"
    ON "agent_evaluations" ("organization_id", "business_unit_id", "suite_run_id", "suite_ordinal")WHERE "suite_run_id" IS NOT NULL;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "fingerprint" TEXT;

--bun:split

ALTER TABLE "assistant_turns" ADD COLUMN "fingerprint" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_evaluation"
    ON "ai_usage_records" ("organization_id", "business_unit_id", "created_at")WHERE "surface" = 'Evaluation';
