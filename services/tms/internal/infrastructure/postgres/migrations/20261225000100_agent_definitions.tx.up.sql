-- An agent definition is an organization's own agent: who it is, what it may
-- do, when it runs, and how much it may do on its own. Templates are only a
-- starting point; nothing about a template bounds the tools an organization
-- may enable.
CREATE TABLE IF NOT EXISTS "agent_definitions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "template" varchar(50),
    "instructions" text,
    "guardrails" text[],
    "tool_names" text[],
    "tool_tiers" jsonb,
    "autonomy_ceiling" varchar(50) NOT NULL DEFAULT 'Propose',
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "shadow_mode" boolean NOT NULL DEFAULT FALSE,
    "decision_timeout_seconds" integer NOT NULL DEFAULT 86400,
    "trigger_mode" varchar(20) NOT NULL DEFAULT 'Chat',
    "cron_expression" varchar(100),
    "cron_timezone" varchar(100),
    "event_kinds" text[],
    "interval_seconds" integer,
    "ends_at" bigint,
    "max_concurrent_runs" integer NOT NULL DEFAULT 1,
    "run_timeout_seconds" integer NOT NULL DEFAULT 600,
    "max_tool_calls" integer NOT NULL DEFAULT 12,
    "context_providers" text[],
    "output_mode" varchar(20) NOT NULL DEFAULT 'Conversational',
    "preferred_provider_id" varchar(100),
    "system_key" varchar(50),
    "last_run_at" bigint,
    "next_run_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
    CONSTRAINT "ck_agent_definitions_instructions_length" CHECK ("instructions" IS NULL OR length("instructions") <= 20000),
    CONSTRAINT "ck_agent_definitions_tool_count" CHECK ("tool_names" IS NULL OR cardinality("tool_names") <= 64)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_org_name" ON "agent_definitions"("organization_id", "business_unit_id", lower("name"));

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_system_key" ON "agent_definitions"("organization_id", "business_unit_id", "system_key")
WHERE
    "system_key" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_lookup" ON "agent_definitions"("organization_id", "business_unit_id", "enabled");

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_due" ON "agent_definitions"("trigger_mode", "enabled", "next_run_at");

--bun:split
COMMENT ON TABLE "agent_definitions" IS 'Per-organization agents: persona, instructions, tools, autonomy and triggers';

COMMENT ON COLUMN "agent_definitions"."template" IS 'The starter this agent was created from, kept for display only';

COMMENT ON COLUMN "agent_definitions"."instructions" IS 'Organization-authored system instructions, placed after the Trenova safety preamble';

COMMENT ON COLUMN "agent_definitions"."tool_names" IS 'Every tool this agent may call, read and write alike; validated against the registry on save';

COMMENT ON COLUMN "agent_definitions"."tool_tiers" IS 'Per-tool autonomy overrides, capped by autonomy_ceiling';

COMMENT ON COLUMN "agent_definitions"."trigger_mode" IS 'Chat, Scheduled (cron), Event (system events) or Continuous (interval)';

COMMENT ON COLUMN "agent_definitions"."system_key" IS 'Set on the agents the platform itself creates and fires; unique per organization';
