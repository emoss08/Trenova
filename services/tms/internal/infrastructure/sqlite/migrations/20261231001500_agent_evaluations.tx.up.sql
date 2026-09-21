-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231001500_agent_evaluations.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_evaluations" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "source_run_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "trigger" TEXT NOT NULL,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "input" TEXT,
    "definition_version" INTEGER NOT NULL DEFAULT 0,
    "prompt_version" TEXT,
    "model" TEXT,
    "provider_id" TEXT,
    "reply" TEXT,
    "actions" TEXT NOT NULL DEFAULT '[]',
    "comparison" TEXT,
    "original_proposals" INTEGER NOT NULL DEFAULT 0,
    "tool_calls_used" INTEGER NOT NULL DEFAULT 0,
    "workflow_id" TEXT,
    "error_message" TEXT,
    "requested_by_user_id" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_evaluations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_evaluations_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Failed')),
    CONSTRAINT "fk_agent_evaluations_run" FOREIGN KEY ("source_run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_evaluations_requested_by" FOREIGN KEY ("requested_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_evaluations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_evaluations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_run"
    ON "agent_evaluations" ("organization_id", "business_unit_id", "source_run_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_definition"
    ON "agent_evaluations" ("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC);
