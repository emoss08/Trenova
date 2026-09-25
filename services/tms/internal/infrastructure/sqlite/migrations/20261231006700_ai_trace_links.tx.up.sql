-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006700_ai_trace_links.tx.up.sql

ALTER TABLE "agent_run_steps" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "agent_run_steps" ADD COLUMN "span_id" TEXT;

--bun:split

ALTER TABLE "agent_run_steps" ADD COLUMN "agent_definition_id" TEXT;

--bun:split

ALTER TABLE "agent_run_steps" ADD COLUMN "agent_definition_version" INTEGER;

--bun:split

ALTER TABLE "agent_run_steps" ADD COLUMN "delegate_call_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "span_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "step_key" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "executed_by_user_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "executed_target_version" INTEGER;

--bun:split

ALTER TABLE "agent_decisions" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "turn_id" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "parent_owner_kind" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "parent_owner_id" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "delegate_call_id" TEXT;

--bun:split

ALTER TABLE "assistant_turns" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "trace_id" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "span_id" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "owner_kind" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "owner_id" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "delegate_call_id" TEXT;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "agent_definition_version" INTEGER;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "attempt" INTEGER;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "failover" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "cache_read_tokens" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "ai_usage_records" ADD COLUMN "cache_write_tokens" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "data_retention" ADD COLUMN "ai_audit_retention_period" INTEGER NOT NULL DEFAULT 2555;
