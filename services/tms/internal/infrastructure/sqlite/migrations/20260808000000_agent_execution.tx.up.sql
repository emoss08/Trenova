-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260808000000_agent_execution.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shadow_mode" INTEGER NOT NULL DEFAULT 1,
    "billing_agent_enabled" INTEGER NOT NULL DEFAULT 0,
    "decision_timeout_seconds" INTEGER NOT NULL DEFAULT 86400,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_controls_decision_timeout" CHECK ("decision_timeout_seconds" >= 60),
    CONSTRAINT "fk_agent_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_controls_tenant" ON "agent_controls" ("organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "agent_runs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_type" TEXT NOT NULL,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "workflow_id" TEXT,
    "model_identifier" TEXT,
    "prompt_version" TEXT NOT NULL,
    "input_context_hash" TEXT NOT NULL,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "error_message" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_runs_subject" ON "agent_runs" ("organization_id", "subject_type", "subject_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_runs_workflow" ON "agent_runs" ("workflow_id");

--bun:split

CREATE TABLE IF NOT EXISTS "agent_proposals"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "tool_name" TEXT NOT NULL,
    "tool_params" TEXT NOT NULL DEFAULT '{}',
    "confidence" NUMERIC NOT NULL DEFAULT 0,
    "rationale" TEXT NOT NULL,
    "evidence" TEXT NOT NULL DEFAULT '[]',
    "autonomy_tier" TEXT NOT NULL DEFAULT 'Propose',
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_proposals" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_proposals_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_run_status" ON "agent_proposals" ("organization_id", "run_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "agent_exceptions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'Medium',
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "attempt_summary" TEXT NOT NULL,
    "evidence" TEXT NOT NULL DEFAULT '[]',
    "blast_radius" INTEGER NOT NULL DEFAULT 0,
    "resolution_state" TEXT NOT NULL DEFAULT 'Open',
    "resolution_notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_exceptions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_exceptions_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_exceptions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_exceptions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_exceptions_queue" ON "agent_exceptions" ("organization_id", "resolution_state", "severity");

--bun:split

CREATE TABLE IF NOT EXISTS "agent_decisions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "proposal_id" TEXT,
    "exception_id" TEXT,
    "decided_by_user_id" TEXT NOT NULL,
    "decision" TEXT NOT NULL,
    "modifications" TEXT,
    "reason_code" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_decisions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_decisions_subject" CHECK (("proposal_id" IS NOT NULL) + ("exception_id" IS NOT NULL) = 1),
    CONSTRAINT "fk_agent_decisions_proposal" FOREIGN KEY ("proposal_id", "business_unit_id", "organization_id") REFERENCES "agent_proposals"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_exception" FOREIGN KEY ("exception_id", "business_unit_id", "organization_id") REFERENCES "agent_exceptions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_user" FOREIGN KEY ("decided_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_decisions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_decisions_proposal" ON "agent_decisions" ("organization_id", "proposal_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_decisions_exception" ON "agent_decisions" ("organization_id", "exception_id");
