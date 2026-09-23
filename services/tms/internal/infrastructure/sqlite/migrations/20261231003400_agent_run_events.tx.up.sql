-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003400_agent_run_events.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_run_events"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "owner_kind" TEXT NOT NULL,
    "owner_id" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL,
    "kind" TEXT NOT NULL,
    "step_key" TEXT NOT NULL DEFAULT '',
    "call_id" TEXT NOT NULL DEFAULT '',
    "payload" TEXT NOT NULL DEFAULT '{}',
    "truncated" INTEGER NOT NULL DEFAULT 0,
    "occurred_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_run_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_run_events_owner_kind" CHECK ("owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_agent_run_events_sequence" CHECK ("sequence" >= 1),
    CONSTRAINT "fk_agent_run_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_run_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_run_events_sequence" ON "agent_run_events" ("organization_id", "owner_id", "sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_run_events_pruning" ON "agent_run_events" ("created_at");
