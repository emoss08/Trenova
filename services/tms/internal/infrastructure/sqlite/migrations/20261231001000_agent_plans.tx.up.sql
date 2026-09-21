-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231001000_agent_plans.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_plans" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "summary" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "step_count" INTEGER NOT NULL DEFAULT 0,
    "completed_steps" INTEGER NOT NULL DEFAULT 0,
    "failed_step" INTEGER,
    "failure_error" TEXT,
    "decided_by_user_id" TEXT,
    "decided_at" INTEGER,
    "expires_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_plans" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_plans_status" CHECK ("status" IN ('Pending', 'Approved', 'Completed', 'Failed', 'Rejected', 'Expired')),
    CONSTRAINT "chk_agent_plans_steps" CHECK ("step_count" >= 0 AND "completed_steps" >= 0 AND "completed_steps" <= "step_count"),
    CONSTRAINT "fk_agent_plans_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_plans_decided_by" FOREIGN KEY ("decided_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_plans_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_plans_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_plans_run" ON "agent_plans" ("organization_id", "run_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_plans_pending" ON "agent_plans" ("organization_id", "status", "expires_at");

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "plan_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "plan_step" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_plan" ON "agent_proposals" ("organization_id", "plan_id", "plan_step")WHERE "plan_id" IS NOT NULL;
