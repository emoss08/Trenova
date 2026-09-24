-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005720_agent_eval_case_retention.tx.up.sql

ALTER TABLE "data_retention" ADD COLUMN "agent_eval_case_retention_period" INTEGER NOT NULL DEFAULT 365;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_retention"
    ON "agent_eval_cases" ("organization_id", "business_unit_id", "created_at");
