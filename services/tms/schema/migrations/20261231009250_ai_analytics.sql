--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Millisecond clocks for the analytics. A call's row is written when the call
-- ends and a step's when it is claimed, so these are the instants a run's
-- duration is measured between. Added without a default first: a volatile
-- default on ADD COLUMN rewrites the table, while SET DEFAULT afterwards only
-- applies to rows written from now on. Rows from before read NULL and fall
-- back to the second-resolution created_at.
ALTER TABLE "ai_usage_records" ADD COLUMN IF NOT EXISTS "recorded_at_ms" BIGINT;

ALTER TABLE "ai_usage_records"
    ALTER COLUMN "recorded_at_ms" SET DEFAULT ((EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT);

ALTER TABLE "agent_run_steps" ADD COLUMN IF NOT EXISTS "started_at_ms" BIGINT;

ALTER TABLE "agent_run_steps"
    ALTER COLUMN "started_at_ms" SET DEFAULT ((EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT);

ALTER TABLE "agent_run_steps" ADD COLUMN IF NOT EXISTS "settled_at_ms" BIGINT;

COMMENT ON COLUMN "ai_usage_records"."recorded_at_ms" IS 'When the attempt ended, in Unix milliseconds; NULL on rows written before it was kept';

COMMENT ON COLUMN "agent_run_steps"."started_at_ms" IS 'When the step was claimed, in Unix milliseconds; NULL on rows written before it was kept';

COMMENT ON COLUMN "agent_run_steps"."settled_at_ms" IS 'When the step''s outcome was recorded, in Unix milliseconds';

-- One finished unit of AI work: an agent run, a conversation turn, or a call a
-- feature made on its own (a document read, a briefing). Worked out from the
-- runs, turns, calls and steps by the analytics refresh, so the analytics read
-- one narrow row per unit instead of joining four tables per page load.
CREATE TABLE IF NOT EXISTS "ai_work_units" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "source" VARCHAR(20) NOT NULL,
    "agent_definition_id" VARCHAR(100),
    "provider_id" VARCHAR(100),
    "model" VARCHAR(200) NOT NULL,
    "model_key" VARCHAR(200) NOT NULL DEFAULT '',
    "task" VARCHAR(100) NOT NULL,
    "outcome" VARCHAR(20) NOT NULL,
    "decision_run_id" VARCHAR(100),
    "started_at_ms" BIGINT NOT NULL,
    "finished_at" BIGINT NOT NULL,
    "duration_ms" BIGINT NOT NULL,
    "first_token_ms" BIGINT,
    "model_calls" INTEGER NOT NULL DEFAULT 0,
    "failed_calls" INTEGER NOT NULL DEFAULT 0,
    "failovers" INTEGER NOT NULL DEFAULT 0,
    "input_tokens" BIGINT NOT NULL DEFAULT 0,
    "output_tokens" BIGINT NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(14, 6),
    "tool_calls" INTEGER NOT NULL DEFAULT 0,
    "tool_failures_model" INTEGER NOT NULL DEFAULT 0,
    "tool_failures_tool" INTEGER NOT NULL DEFAULT 0,
    "refresh_stamp" BIGINT NOT NULL,
    CONSTRAINT "pk_ai_work_units" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_work_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_work_units_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_work_units_source" CHECK ("source" IN ('AgentRun', 'AssistantTurn', 'Call')),
    CONSTRAINT "ck_ai_work_units_outcome" CHECK ("outcome" IN ('Completed', 'HandedOff', 'Stopped', 'Failed')),
    CONSTRAINT "ck_ai_work_units_counts" CHECK (
        "duration_ms" >= 0 AND "model_calls" >= 0 AND "failed_calls" >= 0 AND "failovers" >= 0
        AND "tool_calls" >= 0 AND "tool_failures_model" >= 0 AND "tool_failures_tool" >= 0
    )
);

CREATE INDEX IF NOT EXISTS "idx_ai_work_units_window"
    ON "ai_work_units"("organization_id", "business_unit_id", "finished_at");

CREATE INDEX IF NOT EXISTS "idx_ai_work_units_refresh"
    ON "ai_work_units"("organization_id", "business_unit_id", "refresh_stamp");

CREATE INDEX IF NOT EXISTS "idx_ai_work_units_decision_run"
    ON "ai_work_units"("organization_id", "business_unit_id", "decision_run_id")
    WHERE "decision_run_id" IS NOT NULL;

COMMENT ON TABLE "ai_work_units" IS 'One finished unit of AI work (agent run, conversation turn or standalone call) with the model that served it, how it ended, how long it took, what it cost and how its tool calls went; rebuilt by the analytics refresh';

-- A unit's tool calls by tool and by why the failed ones failed. An empty
-- cause is the calls that ran.
CREATE TABLE IF NOT EXISTS "ai_work_unit_tools" (
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "unit_id" VARCHAR(100) NOT NULL,
    "tool_name" VARCHAR(200) NOT NULL,
    "cause" VARCHAR(30) NOT NULL DEFAULT '',
    "finished_at" BIGINT NOT NULL,
    "model_key" VARCHAR(200) NOT NULL DEFAULT '',
    "provider_id" VARCHAR(100) NOT NULL DEFAULT '',
    "agent_definition_id" VARCHAR(100) NOT NULL DEFAULT '',
    "task" VARCHAR(100) NOT NULL DEFAULT '',
    "calls" INTEGER NOT NULL,
    CONSTRAINT "pk_ai_work_unit_tools" PRIMARY KEY ("organization_id", "business_unit_id", "unit_id", "tool_name", "cause"),
    CONSTRAINT "fk_ai_work_unit_tools_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_work_unit_tools_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_work_unit_tools_calls" CHECK ("calls" > 0),
    CONSTRAINT "ck_ai_work_unit_tools_cause" CHECK ("cause" IN (
        '', 'WrongArguments', 'ToolTimedOut', 'RepeatedCall', 'BlockedByRule', 'ToolError', 'ToolNotHeld'
    ))
);

CREATE INDEX IF NOT EXISTS "idx_ai_work_unit_tools_window"
    ON "ai_work_unit_tools"("organization_id", "business_unit_id", "finished_at");

COMMENT ON TABLE "ai_work_unit_tools" IS 'Tool calls per unit of AI work, by tool and failure cause (empty for calls that ran); rebuilt with the unit, whose model, provider, agent and task it carries so tool reads need no join';

-- Calls per provider, model, task, agent and error class per UTC hour, with
-- the first-token latencies of the calls that answered as a fixed log-scale
-- histogram so any span of hours can be merged and its percentiles read.
CREATE TABLE IF NOT EXISTS "ai_provider_hourly_stats" (
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "hour" BIGINT NOT NULL,
    "provider_id" VARCHAR(100) NOT NULL DEFAULT '',
    "model" VARCHAR(200) NOT NULL,
    "model_key" VARCHAR(200) NOT NULL DEFAULT '',
    "task" VARCHAR(100) NOT NULL,
    "agent_definition_id" VARCHAR(100) NOT NULL DEFAULT '',
    "error_class" VARCHAR(50) NOT NULL DEFAULT '',
    "calls" INTEGER NOT NULL,
    "fell_to_next" INTEGER NOT NULL DEFAULT 0,
    "first_token_hist" INTEGER[] NOT NULL DEFAULT '{}',
    CONSTRAINT "pk_ai_provider_hourly_stats" PRIMARY KEY (
        "organization_id", "business_unit_id", "hour", "provider_id", "model", "task", "agent_definition_id", "error_class"
    ),
    CONSTRAINT "fk_ai_provider_hourly_stats_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_provider_hourly_stats_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_provider_hourly_stats_counts" CHECK ("calls" > 0 AND "fell_to_next" >= 0 AND "fell_to_next" <= "calls"),
    CONSTRAINT "ck_ai_provider_hourly_stats_error_class" CHECK ("error_class" IN (
        '', 'RateLimited', 'TimedOut', 'CouldNotConnect', 'AccessDenied', 'ServerError', 'Declined', 'Cancelled'
    ))
);

COMMENT ON TABLE "ai_provider_hourly_stats" IS 'Model calls per UTC hour by provider, model, task, agent and provider error (empty for answered calls; Declined and Cancelled are not the provider''s fault), with how many fell through to the next provider and a first-token histogram; rebuilt by the analytics refresh';

-- How far each organization's analytics have been worked out.
CREATE TABLE IF NOT EXISTS "ai_analytics_refresh_state" (
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "refreshed_through" BIGINT NOT NULL,
    "refreshed_at" BIGINT NOT NULL,
    CONSTRAINT "pk_ai_analytics_refresh_state" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_ai_analytics_refresh_state_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_analytics_refresh_state_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "ai_analytics_refresh_state" IS 'The instant each organization''s AI analytics are worked out through, and when that last happened';

SELECT trenova_rls.reconcile();
