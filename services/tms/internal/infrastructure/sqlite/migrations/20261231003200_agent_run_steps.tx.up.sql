-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003200_agent_run_steps.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_run_steps"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "owner_kind" TEXT NOT NULL,
    "owner_id" TEXT NOT NULL,
    "attempt" INTEGER NOT NULL DEFAULT 1,
    "kind" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "step_key" TEXT NOT NULL,
    "tool_name" TEXT NOT NULL DEFAULT '',
    "call_id" TEXT NOT NULL DEFAULT '',
    "arguments" TEXT NOT NULL DEFAULT '{}',
    "outcome" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_run_steps" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_run_steps_owner_kind" CHECK ("owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_agent_run_steps_kind" CHECK ("kind" IN ('Tool', 'Completion')),
    CONSTRAINT "ck_agent_run_steps_status" CHECK ("status" IN ('Started', 'Completed', 'Failed')),
    CONSTRAINT "fk_agent_run_steps_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_run_steps_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_run_steps_key" ON "agent_run_steps" ("organization_id", "owner_id", "step_key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_run_steps_owner" ON "agent_run_steps" ("organization_id", "business_unit_id", "owner_id", "created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_run_steps_pruning" ON "agent_run_steps" ("created_at");
