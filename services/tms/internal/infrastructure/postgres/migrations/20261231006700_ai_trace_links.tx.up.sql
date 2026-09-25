-- What the AI audit trail and its traces are read from. Every column is
-- nullable or defaulted, so a row written before the release stays valid and
-- the runtime fills them only once it knows them. The indexes the audit
-- projector scans by are built without a lock in the next migration.
ALTER TABLE "agent_run_steps"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32),
    ADD COLUMN IF NOT EXISTS "span_id" varchar(16),
    ADD COLUMN IF NOT EXISTS "agent_definition_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "agent_definition_version" bigint,
    ADD COLUMN IF NOT EXISTS "delegate_call_id" varchar(200);

COMMENT ON COLUMN "agent_run_steps"."trace_id" IS 'The trace the step ran in, as 32 lowercase hex characters';

COMMENT ON COLUMN "agent_run_steps"."span_id" IS 'The tool span the step ran in, as 16 lowercase hex characters';

COMMENT ON COLUMN "agent_run_steps"."agent_definition_id" IS 'The agent that made the call, which for a delegate''s step is not the agent the run belongs to';

COMMENT ON COLUMN "agent_run_steps"."agent_definition_version" IS 'The version of that agent''s definition when the call was made';

COMMENT ON COLUMN "agent_run_steps"."delegate_call_id" IS 'The delegate_task call whose task the step was part of; null for the owner''s own steps';

--bun:split

ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32),
    ADD COLUMN IF NOT EXISTS "span_id" varchar(16),
    ADD COLUMN IF NOT EXISTS "step_key" varchar(120),
    ADD COLUMN IF NOT EXISTS "executed_by_user_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "executed_target_version" bigint;

COMMENT ON COLUMN "agent_proposals"."trace_id" IS 'The trace of the tool call that raised the proposal, so a decision made later links back to it';

COMMENT ON COLUMN "agent_proposals"."span_id" IS 'The tool span that raised the proposal';

COMMENT ON COLUMN "agent_proposals"."step_key" IS 'The agent_run_steps key of the call that raised the proposal';

COMMENT ON COLUMN "agent_proposals"."executed_by_user_id" IS 'The person the write ran as once it ran; null for a write no person stood behind';

COMMENT ON COLUMN "agent_proposals"."executed_target_version" IS 'The target record''s version after the write, when the tool reports it';

--bun:split

ALTER TABLE "agent_decisions"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32);

COMMENT ON COLUMN "agent_decisions"."trace_id" IS 'The trace the decision was made in';

--bun:split

ALTER TABLE "agent_runs"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32),
    ADD COLUMN IF NOT EXISTS "turn_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "parent_owner_kind" varchar(20),
    ADD COLUMN IF NOT EXISTS "parent_owner_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "delegate_call_id" varchar(200);

COMMENT ON COLUMN "agent_runs"."trace_id" IS 'The run''s trace, derived from the run so it is known before the run starts';

COMMENT ON COLUMN "agent_runs"."turn_id" IS 'The conversation turn the run was opened for, when one was';

COMMENT ON COLUMN "agent_runs"."parent_owner_kind" IS 'For a delegate''s run, what handed it the task: AgentRun or AssistantTurn';

COMMENT ON COLUMN "agent_runs"."parent_owner_id" IS 'For a delegate''s run, the run or turn that handed it the task';

COMMENT ON COLUMN "agent_runs"."delegate_call_id" IS 'For a delegate''s run, the delegate_task call that handed it the task';

--bun:split

ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_kind";

--bun:split

ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_parent_owner_kind" CHECK (
        "parent_owner_kind" IS NULL OR "parent_owner_kind" IN ('AgentRun', 'AssistantTurn')
    ) NOT VALID;

--bun:split

ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_pair";

--bun:split

ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_parent_owner_pair" CHECK (
        ("parent_owner_kind" IS NULL) = ("parent_owner_id" IS NULL)
        AND ("delegate_call_id" IS NULL OR "parent_owner_id" IS NOT NULL)
    ) NOT VALID;

--bun:split

ALTER TABLE "assistant_turns"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32);

COMMENT ON COLUMN "assistant_turns"."trace_id" IS 'The turn''s trace, derived from the turn so it is known before the turn starts';

--bun:split

ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "trace_id" varchar(32),
    ADD COLUMN IF NOT EXISTS "span_id" varchar(16),
    ADD COLUMN IF NOT EXISTS "owner_kind" varchar(20),
    ADD COLUMN IF NOT EXISTS "owner_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "delegate_call_id" varchar(200),
    ADD COLUMN IF NOT EXISTS "agent_definition_version" bigint,
    ADD COLUMN IF NOT EXISTS "attempt" integer,
    ADD COLUMN IF NOT EXISTS "failover" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "cache_read_tokens" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "cache_write_tokens" integer NOT NULL DEFAULT 0;

COMMENT ON COLUMN "ai_usage_records"."trace_id" IS 'The trace the attempt ran in';

COMMENT ON COLUMN "ai_usage_records"."span_id" IS 'The span of the attempt';

COMMENT ON COLUMN "ai_usage_records"."owner_kind" IS 'What the call was made for: AgentRun or AssistantTurn; null for work that is neither';

COMMENT ON COLUMN "ai_usage_records"."owner_id" IS 'The run or turn the call was made for';

COMMENT ON COLUMN "ai_usage_records"."delegate_call_id" IS 'The delegate_task call whose task the call was part of';

COMMENT ON COLUMN "ai_usage_records"."agent_definition_version" IS 'The version of the agent''s definition when the call was made';

COMMENT ON COLUMN "ai_usage_records"."attempt" IS 'Which try of the call this was, from 1; null for a row written before it was kept';

COMMENT ON COLUMN "ai_usage_records"."failover" IS 'The attempt went to a provider other than the first one tried';

COMMENT ON COLUMN "ai_usage_records"."cache_read_tokens" IS 'Input tokens the provider served from its prompt cache';

COMMENT ON COLUMN "ai_usage_records"."cache_write_tokens" IS 'Input tokens the provider wrote to its prompt cache';

--bun:split

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_owner_kind";

--bun:split

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_owner_kind" CHECK (
        "owner_kind" IS NULL OR "owner_kind" IN ('AgentRun', 'AssistantTurn')
    ) NOT VALID;

--bun:split

ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "ai_audit_retention_period" integer NOT NULL DEFAULT 2555;

COMMENT ON COLUMN "data_retention"."ai_audit_retention_period" IS 'Days the AI audit trail is kept before it is pruned; seven years by default and never less than one';

--bun:split

ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_audit_retention_period";

--bun:split

ALTER TABLE "data_retention"
    ADD CONSTRAINT "ck_data_retention_ai_audit_retention_period" CHECK ("ai_audit_retention_period" >= 365);
