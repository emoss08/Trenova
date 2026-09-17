-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261225000100_agent_definitions.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_definitions"(
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
    CONSTRAINT "pk_agent_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_definitions_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definitions_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_definitions_template" CHECK ("template" IS NULL OR "template" IN ('DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant', 'GeneralAssistant', 'BillingException', 'DispatchAssignment', 'ImportAssistant')),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_org_name" ON "agent_definitions" ("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_system_key" ON "agent_definitions" ("organization_id", "business_unit_id", "system_key")WHERE
    "system_key" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_lookup" ON "agent_definitions" ("organization_id", "business_unit_id", "enabled");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_due" ON "agent_definitions" ("trigger_mode", "enabled", "next_run_at");
