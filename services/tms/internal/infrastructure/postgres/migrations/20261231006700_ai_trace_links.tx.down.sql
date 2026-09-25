ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_audit_retention_period";

--bun:split

ALTER TABLE "data_retention"
    DROP COLUMN IF EXISTS "ai_audit_retention_period";

--bun:split

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_owner_kind";

--bun:split

ALTER TABLE "ai_usage_records"
    DROP COLUMN IF EXISTS "cache_write_tokens",
    DROP COLUMN IF EXISTS "cache_read_tokens",
    DROP COLUMN IF EXISTS "failover",
    DROP COLUMN IF EXISTS "attempt",
    DROP COLUMN IF EXISTS "agent_definition_version",
    DROP COLUMN IF EXISTS "delegate_call_id",
    DROP COLUMN IF EXISTS "owner_id",
    DROP COLUMN IF EXISTS "owner_kind",
    DROP COLUMN IF EXISTS "span_id",
    DROP COLUMN IF EXISTS "trace_id";

--bun:split

ALTER TABLE "assistant_turns"
    DROP COLUMN IF EXISTS "trace_id";

--bun:split

ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_pair";

--bun:split

ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_kind";

--bun:split

ALTER TABLE "agent_runs"
    DROP COLUMN IF EXISTS "delegate_call_id",
    DROP COLUMN IF EXISTS "parent_owner_id",
    DROP COLUMN IF EXISTS "parent_owner_kind",
    DROP COLUMN IF EXISTS "turn_id",
    DROP COLUMN IF EXISTS "trace_id";

--bun:split

ALTER TABLE "agent_decisions"
    DROP COLUMN IF EXISTS "trace_id";

--bun:split

ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "executed_target_version",
    DROP COLUMN IF EXISTS "executed_by_user_id",
    DROP COLUMN IF EXISTS "step_key",
    DROP COLUMN IF EXISTS "span_id",
    DROP COLUMN IF EXISTS "trace_id";

--bun:split

ALTER TABLE "agent_run_steps"
    DROP COLUMN IF EXISTS "delegate_call_id",
    DROP COLUMN IF EXISTS "agent_definition_version",
    DROP COLUMN IF EXISTS "agent_definition_id",
    DROP COLUMN IF EXISTS "span_id",
    DROP COLUMN IF EXISTS "trace_id";
