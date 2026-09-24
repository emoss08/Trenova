-- A suite run is one agent answering a sample of its active evaluation cases,
-- scored as a whole and compared with the runs before it. The nightly sweep
-- opens one per agent whose fingerprint or cases changed; an administrator can
-- open one at any time.
CREATE TABLE IF NOT EXISTS "agent_suite_runs" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "trigger" varchar(20) NOT NULL DEFAULT 'Scheduled',
    "sweep_key" varchar(200),
    "fingerprint" jsonb,
    "fingerprint_hash" varchar(64) NOT NULL,
    "fingerprint_changes" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "suite_revision" varchar(64) NOT NULL,
    "sample_seed" bigint NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Running',
    "cases_total" integer NOT NULL DEFAULT 0,
    "cases_passed" integer NOT NULL DEFAULT 0,
    "cases_failed" integer NOT NULL DEFAULT 0,
    "cases_skipped" integer NOT NULL DEFAULT 0,
    "hard_failures" integer NOT NULL DEFAULT 0,
    "deterministic_score" double precision,
    "judge_score" double precision,
    "quality_score" double precision,
    "baseline_score" double precision,
    "baseline_run_id" varchar(100),
    "regression" boolean NOT NULL DEFAULT false,
    "cost_usd" numeric(14, 6) NOT NULL DEFAULT 0,
    "started_at" bigint NOT NULL,
    "finished_at" bigint,
    "comments" text,
    "workflow_id" varchar(255),
    "requested_by_user_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    ON "agent_suite_runs"("organization_id", "business_unit_id", "sweep_key")
    WHERE "sweep_key" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_suite_runs_agent"
    ON "agent_suite_runs"("organization_id", "business_unit_id", "agent_definition_id", "started_at" DESC, "id" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_suite_runs_started"
    ON "agent_suite_runs"("organization_id", "business_unit_id", "started_at" DESC, "id" DESC);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_suite_runs_one_running"
    ON "agent_suite_runs"("organization_id", "business_unit_id", "agent_definition_id")
    WHERE "status" = 'Running';

COMMENT ON TABLE "agent_suite_runs" IS 'One agent answering a sample of its active evaluation cases, scored as a whole and compared with the runs before it';

COMMENT ON COLUMN "agent_suite_runs"."agent_definition_id" IS 'The agent the suite scored';

COMMENT ON COLUMN "agent_suite_runs"."trigger" IS 'Scheduled for the nightly sweep, Manual for a run an administrator started';

COMMENT ON COLUMN "agent_suite_runs"."sweep_key" IS 'The sweep execution and agent the run was opened for, so a retried plan opens it once';

COMMENT ON COLUMN "agent_suite_runs"."fingerprint" IS 'The agent as it was scored: definition version, prompt hash over a fixed context, tool spec hash, model and provider';

COMMENT ON COLUMN "agent_suite_runs"."fingerprint_hash" IS 'SHA-256 of the fingerprint; a run with the same hash and suite revision as the last completed one is skipped';

COMMENT ON COLUMN "agent_suite_runs"."fingerprint_changes" IS 'What changed in the fingerprint since the run it is compared with';

COMMENT ON COLUMN "agent_suite_runs"."suite_revision" IS 'SHA-256 of the active cases and their versions when the run was planned';

COMMENT ON COLUMN "agent_suite_runs"."sample_seed" IS 'The seed the cases were drawn with, derived from the run id';

COMMENT ON COLUMN "agent_suite_runs"."status" IS 'Running, Completed, Skipped when nothing changed, BudgetStopped when the nightly or monthly evaluation budget ran out, or Failed';

COMMENT ON COLUMN "agent_suite_runs"."cases_total" IS 'How many cases the run drew';

COMMENT ON COLUMN "agent_suite_runs"."cases_passed" IS 'Cases whose final score reached the pass mark with no hard check broken';

COMMENT ON COLUMN "agent_suite_runs"."cases_failed" IS 'Cases that broke a hard check or scored below the pass mark, or whose replay failed';

COMMENT ON COLUMN "agent_suite_runs"."cases_skipped" IS 'Cases that were not replayed: stopped by the budget, or unable to run as captured';

COMMENT ON COLUMN "agent_suite_runs"."hard_failures" IS 'Cases that broke a hard check';

COMMENT ON COLUMN "agent_suite_runs"."deterministic_score" IS 'The weighted mean of the cases'' deterministic scores, before any judge';

COMMENT ON COLUMN "agent_suite_runs"."judge_score" IS 'The mean of the judge''s scores over the cases it read, when judging is on';

COMMENT ON COLUMN "agent_suite_runs"."quality_score" IS 'The weighted mean of the cases'' final scores: deterministic blended with the judge, zero on a hard failure';

COMMENT ON COLUMN "agent_suite_runs"."baseline_score" IS 'The median quality score of the recent runs this one is compared with';

COMMENT ON COLUMN "agent_suite_runs"."baseline_run_id" IS 'The most recent completed run before this one, whose fingerprint the changes are measured against';

COMMENT ON COLUMN "agent_suite_runs"."regression" IS 'Whether the run regressed: its score fell further below the baseline than the threshold, or a case that passed at the baseline broke a hard check';

COMMENT ON COLUMN "agent_suite_runs"."cost_usd" IS 'What the run''s replays and judgements cost at their providers'' configured prices';

COMMENT ON COLUMN "agent_suite_runs"."started_at" IS 'When the run was opened';

COMMENT ON COLUMN "agent_suite_runs"."finished_at" IS 'When the run was scored, skipped, stopped or failed';

COMMENT ON COLUMN "agent_suite_runs"."comments" IS 'Why the run was skipped, stopped or failed, or what regressed';

COMMENT ON COLUMN "agent_suite_runs"."workflow_id" IS 'The workflow that replays the run''s cases';

COMMENT ON COLUMN "agent_suite_runs"."requested_by_user_id" IS 'The administrator who started the run; empty for the nightly sweep';

--bun:split
-- One row per organization says when and how far the nightly sweep runs. An
-- organization without a row runs on the defaults.
CREATE TABLE IF NOT EXISTS "agent_quality_controls" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "enabled" boolean NOT NULL DEFAULT true,
    "run_hour_local" smallint NOT NULL DEFAULT 2,
    "timezone" varchar(100),
    "max_cases_per_agent" integer NOT NULL DEFAULT 50,
    "nightly_budget_usd" numeric(14, 2) NOT NULL DEFAULT 5.00,
    "monthly_budget_usd" numeric(14, 2) NOT NULL DEFAULT 50.00,
    "judge_enabled" boolean NOT NULL DEFAULT false,
    "judge_sample_rate" double precision NOT NULL DEFAULT 0.2,
    "regression_threshold" double precision NOT NULL DEFAULT 0.10,
    "min_cases" integer NOT NULL DEFAULT 10,
    "force_rerun_days" integer NOT NULL DEFAULT 7,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    ON "agent_quality_controls"("organization_id", "business_unit_id");

COMMENT ON TABLE "agent_quality_controls" IS 'When and how far the nightly agent quality sweep runs for an organization';

COMMENT ON COLUMN "agent_quality_controls"."enabled" IS 'Whether the nightly sweep runs at all; a suite can still be run by hand';

COMMENT ON COLUMN "agent_quality_controls"."run_hour_local" IS 'The hour of the night the sweep starts, in the timezone below';

COMMENT ON COLUMN "agent_quality_controls"."timezone" IS 'The timezone the hour is read in; empty for the organization''s own';

COMMENT ON COLUMN "agent_quality_controls"."max_cases_per_agent" IS 'The most cases one agent''s suite run draws';

COMMENT ON COLUMN "agent_quality_controls"."nightly_budget_usd" IS 'The most evaluation may spend in one day, in US dollars at the providers'' configured prices';

COMMENT ON COLUMN "agent_quality_controls"."monthly_budget_usd" IS 'The most evaluation may spend in one calendar month';

COMMENT ON COLUMN "agent_quality_controls"."judge_enabled" IS 'Whether a judge model reads a sample of each suite''s answers';

COMMENT ON COLUMN "agent_quality_controls"."judge_sample_rate" IS 'The share of each suite''s cases the judge reads, from 0 to 1';

COMMENT ON COLUMN "agent_quality_controls"."regression_threshold" IS 'How far below the recent median a suite score must fall to count as a regression';

COMMENT ON COLUMN "agent_quality_controls"."min_cases" IS 'The fewest cases a run must score before a fall in its score counts as a regression';

COMMENT ON COLUMN "agent_quality_controls"."force_rerun_days" IS 'An unchanged agent is run again once its last run is this many days old';

--bun:split
ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "suite_run_id" varchar(100);

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "suite_ordinal" integer;

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "fk_agent_evaluations_suite_run";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "fk_agent_evaluations_suite_run" FOREIGN KEY ("suite_run_id", "business_unit_id", "organization_id") REFERENCES "agent_suite_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE;

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_suite_ordinal";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "ck_agent_evaluations_suite_ordinal" CHECK (("suite_run_id" IS NULL) = ("suite_ordinal" IS NULL));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_evaluations_suite_ordinal"
    ON "agent_evaluations"("organization_id", "business_unit_id", "suite_run_id", "suite_ordinal")
    WHERE "suite_run_id" IS NOT NULL;

COMMENT ON COLUMN "agent_evaluations"."suite_run_id" IS 'The suite run the replay belongs to; empty for a replay started on its own';

COMMENT ON COLUMN "agent_evaluations"."suite_ordinal" IS 'The replay''s place in its suite run, which is the order the cases are asked in';

--bun:split
ALTER TABLE "agent_runs"
    ADD COLUMN IF NOT EXISTS "fingerprint" jsonb;

COMMENT ON COLUMN "agent_runs"."fingerprint" IS 'The agent as the run opened: definition version, prompt hash over a fixed context, tool spec hash, and the model and provider that answered';

--bun:split
ALTER TABLE "assistant_turns"
    ADD COLUMN IF NOT EXISTS "fingerprint" jsonb;

COMMENT ON COLUMN "assistant_turns"."fingerprint" IS 'The agent as the turn opened: definition version, prompt hash over a fixed context, tool spec hash, and the model and provider that answered';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_evaluation"
    ON "ai_usage_records"("organization_id", "business_unit_id", "created_at")
    WHERE "surface" = 'Evaluation';
