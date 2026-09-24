-- Hand-written: SQLite cannot alter a CHECK constraint, so agent_definitions
-- is rebuilt with the widened template check and its rows copied across, as
-- 20261231003700 did. Four tables reference it with ON DELETE CASCADE, and
-- dropping the old table would empty them through that cascade when foreign
-- keys are enforced, so their rows are set aside and cleared while the old
-- table still stands, then put back once the rebuilt table holds every agent
-- again. The children keep their own definitions and indexes: they name
-- agent_definitions, which exists again by the time a row is written back.
-- Source: 20261231006500_agent_definition_event_templates.tx.up.sql

CREATE TEMP TABLE "agent_tool_trust_backup" AS SELECT * FROM "agent_tool_trust";

--bun:split

CREATE TEMP TABLE "role_agent_grants_backup" AS SELECT * FROM "role_agent_grants";

--bun:split

CREATE TEMP TABLE "agent_eval_cases_backup" AS SELECT * FROM "agent_eval_cases";

--bun:split

CREATE TEMP TABLE "agent_suite_runs_backup" AS SELECT * FROM "agent_suite_runs";

--bun:split

CREATE TABLE "agent_definitions_rebuilt"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "template" TEXT,
    "instructions" TEXT,
    "guardrails" TEXT,
    "tool_names" TEXT,
    "tool_tiers" TEXT,
    "autonomy_ceiling" TEXT NOT NULL DEFAULT 'Propose',
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "shadow_mode" INTEGER NOT NULL DEFAULT 0,
    "decision_timeout_seconds" INTEGER NOT NULL DEFAULT 86400,
    "trigger_mode" TEXT NOT NULL DEFAULT 'Chat',
    "cron_expression" TEXT,
    "cron_timezone" TEXT,
    "event_kinds" TEXT,
    "interval_seconds" INTEGER,
    "ends_at" INTEGER,
    "max_concurrent_runs" INTEGER NOT NULL DEFAULT 1,
    "run_timeout_seconds" INTEGER NOT NULL DEFAULT 600,
    "max_tool_calls" INTEGER NOT NULL DEFAULT 12,
    "context_providers" TEXT,
    "output_mode" TEXT NOT NULL DEFAULT 'Conversational',
    "preferred_provider_id" TEXT,
    "system_key" TEXT,
    "last_run_at" INTEGER,
    "next_run_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "icon" TEXT,
    "accent" TEXT,
    "monthly_budget_usd" REAL,
    "daily_run_limit" INTEGER NOT NULL DEFAULT 0,
    "tool_daily_limits" TEXT NOT NULL DEFAULT '{}',
    "simulation_mode" INTEGER NOT NULL DEFAULT 0,
    "delegate_ids" TEXT,
    "access_mode" TEXT NOT NULL DEFAULT 'Everyone',
    "memory_token_budget" INTEGER,
    "data_access_ceiling" TEXT NOT NULL DEFAULT 'Internal',
    CONSTRAINT "pk_agent_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_definitions_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definitions_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_definitions_template" CHECK ("template" IS NULL OR "template" IN (
        'DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant',
        'GeneralAssistant', 'BillingException', 'DispatchAssignment', 'ImportAssistant',
        'LoadMonitor', 'ShipmentIntake', 'CashApplication', 'DetentionDesk',
        'CredentialDesk', 'CustomerUpdateDesk', 'CarrierRiskDesk', 'IntakeDesk',
        'LoadEntryCheck', 'ServiceFailureDesk', 'InsightAnalyst', 'EDIDesk',
        'FormulaAssistant'
    )),
    CONSTRAINT "ck_agent_definitions_autonomy" CHECK ("autonomy_ceiling" IN ('Propose', 'ActWithApproval', 'AutoExecute')),
    CONSTRAINT "ck_agent_definitions_trigger_mode" CHECK ("trigger_mode" IN ('Chat', 'Scheduled', 'Event', 'Continuous')),
    CONSTRAINT "ck_agent_definitions_output_mode" CHECK ("output_mode" IN ('Conversational', 'Report')),
    CONSTRAINT "ck_agent_definitions_cron" CHECK ("trigger_mode" <> 'Scheduled' OR "cron_expression" IS NOT NULL),
    CONSTRAINT "ck_agent_definitions_interval" CHECK ("trigger_mode" <> 'Continuous' OR "interval_seconds" >= 60),
    CONSTRAINT "ck_agent_definitions_decision_timeout" CHECK ("decision_timeout_seconds" >= 60),
    CONSTRAINT "ck_agent_definitions_run_timeout" CHECK ("run_timeout_seconds" >= 60),
    CONSTRAINT "ck_agent_definitions_tool_calls" CHECK ("max_tool_calls" BETWEEN 1 AND 64),
    CONSTRAINT "ck_agent_definitions_concurrency" CHECK ("max_concurrent_runs" BETWEEN 1 AND 10),
    CONSTRAINT "ck_agent_definitions_instructions_length" CHECK ("instructions" IS NULL OR length("instructions") <= 20000)
);

--bun:split

INSERT INTO "agent_definitions_rebuilt" (
    "id", "business_unit_id", "organization_id", "name", "description", "template", "instructions",
    "guardrails", "tool_names", "tool_tiers", "autonomy_ceiling", "enabled", "shadow_mode",
    "decision_timeout_seconds", "trigger_mode", "cron_expression", "cron_timezone", "event_kinds",
    "interval_seconds", "ends_at", "max_concurrent_runs", "run_timeout_seconds", "max_tool_calls",
    "context_providers", "output_mode", "preferred_provider_id", "system_key", "last_run_at",
    "next_run_at", "version", "created_at", "updated_at", "icon", "accent", "monthly_budget_usd",
    "daily_run_limit", "tool_daily_limits", "simulation_mode", "delegate_ids", "access_mode",
    "memory_token_budget", "data_access_ceiling"
)
SELECT
    "id", "business_unit_id", "organization_id", "name", "description", "template", "instructions",
    "guardrails", "tool_names", "tool_tiers", "autonomy_ceiling", "enabled", "shadow_mode",
    "decision_timeout_seconds", "trigger_mode", "cron_expression", "cron_timezone", "event_kinds",
    "interval_seconds", "ends_at", "max_concurrent_runs", "run_timeout_seconds", "max_tool_calls",
    "context_providers", "output_mode", "preferred_provider_id", "system_key", "last_run_at",
    "next_run_at", "version", "created_at", "updated_at", "icon", "accent", "monthly_budget_usd",
    "daily_run_limit", "tool_daily_limits", "simulation_mode", "delegate_ids", "access_mode",
    "memory_token_budget", "data_access_ceiling"
FROM "agent_definitions";

--bun:split

DELETE FROM "agent_tool_trust";

--bun:split

DELETE FROM "role_agent_grants";

--bun:split

DELETE FROM "agent_eval_cases";

--bun:split

DELETE FROM "agent_suite_runs";

--bun:split

DROP TABLE "agent_definitions";

--bun:split

ALTER TABLE "agent_definitions_rebuilt" RENAME TO "agent_definitions";

--bun:split

CREATE UNIQUE INDEX "uq_agent_definitions_org_name" ON "agent_definitions" ("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE UNIQUE INDEX "uq_agent_definitions_system_key" ON "agent_definitions" ("organization_id", "business_unit_id", "system_key") WHERE
    "system_key" IS NOT NULL;

--bun:split

CREATE INDEX "idx_agent_definitions_lookup" ON "agent_definitions" ("organization_id", "business_unit_id", "enabled");

--bun:split

CREATE INDEX "idx_agent_definitions_due" ON "agent_definitions" ("trigger_mode", "enabled", "next_run_at");

--bun:split

INSERT INTO "agent_tool_trust" SELECT * FROM "agent_tool_trust_backup";

--bun:split

INSERT INTO "role_agent_grants" SELECT * FROM "role_agent_grants_backup";

--bun:split

INSERT INTO "agent_eval_cases" SELECT * FROM "agent_eval_cases_backup";

--bun:split

INSERT INTO "agent_suite_runs" SELECT * FROM "agent_suite_runs_backup";

--bun:split

DROP TABLE "agent_tool_trust_backup";

--bun:split

DROP TABLE "role_agent_grants_backup";

--bun:split

DROP TABLE "agent_eval_cases_backup";

--bun:split

DROP TABLE "agent_suite_runs_backup";
