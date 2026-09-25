-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006710_ai_trace_link_indexes.up.sql

CREATE INDEX IF NOT EXISTS "idx_agent_run_steps_updated" ON "agent_run_steps" ("updated_at", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_updated" ON "agent_proposals" ("updated_at", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_decisions_created" ON "agent_decisions" ("created_at", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_runs_updated" ON "agent_runs" ("updated_at", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_turns_updated" ON "assistant_turns" ("updated_at", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_created" ON "ai_usage_records" ("created_at", "id");
