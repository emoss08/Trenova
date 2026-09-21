-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231001300_agent_budgets.tx.up.sql

ALTER TABLE "agent_definitions" ADD COLUMN "monthly_budget_usd" REAL;

--bun:split

ALTER TABLE "agent_definitions" ADD COLUMN "daily_run_limit" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_definitions" ADD COLUMN "tool_daily_limits" TEXT NOT NULL DEFAULT '{}';

--bun:split

ALTER TABLE "agent_definitions" ADD COLUMN "simulation_mode" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "simulated_at" INTEGER;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "simulation" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_executed_tool"
    ON "agent_proposals" ("organization_id", "business_unit_id", "tool_name", "executed_at")WHERE "executed_at" IS NOT NULL;
