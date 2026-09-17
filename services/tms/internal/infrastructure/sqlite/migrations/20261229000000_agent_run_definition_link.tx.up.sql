-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261229000000_agent_run_definition_link.tx.up.sql

ALTER TABLE "agent_runs" ADD COLUMN "agent_definition_id" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "trigger" TEXT NOT NULL DEFAULT 'Manual';

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "summary" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_runs_definition" ON "agent_runs" ("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC);
