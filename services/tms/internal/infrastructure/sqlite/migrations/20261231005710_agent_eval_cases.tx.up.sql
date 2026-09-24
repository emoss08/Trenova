-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005710_agent_eval_cases.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_eval_cases" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "title" TEXT,
    "source" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Candidate',
    "trigger" TEXT NOT NULL,
    "source_run_id" TEXT,
    "source_turn_id" TEXT,
    "source_thread_id" TEXT,
    "source_message_id" TEXT,
    "source_proposal_id" TEXT,
    "source_feedback_id" TEXT,
    "input" TEXT NOT NULL,
    "history" TEXT NOT NULL DEFAULT '[]',
    "page_context" TEXT,
    "mentions" TEXT NOT NULL DEFAULT '[]',
    "subject_type" TEXT,
    "subject_id" TEXT,
    "held_tools" TEXT NOT NULL DEFAULT '[]',
    "tool_fixtures" TEXT NOT NULL DEFAULT '[]',
    "expected" TEXT NOT NULL DEFAULT '{}',
    "rubric" TEXT,
    "redaction" TEXT,
    "content_hash" TEXT NOT NULL,
    "captured_fingerprint" TEXT,
    "expires_at" INTEGER,
    "created_by_user_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_eval_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_eval_cases_source" CHECK ("source" IN ('DecidedProposal', 'ThumbsUp', 'Curated')),
    CONSTRAINT "ck_agent_eval_cases_status" CHECK ("status" IN ('Candidate', 'Active', 'Quarantined', 'Retired')),
    CONSTRAINT "ck_agent_eval_cases_subject" CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL)),
    CONSTRAINT "fk_agent_eval_cases_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_eval_cases_feedback" FOREIGN KEY ("source_feedback_id", "business_unit_id", "organization_id") REFERENCES "ai_feedback"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_eval_cases_created_by" FOREIGN KEY ("created_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_eval_cases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_eval_cases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_eval_cases_content"
    ON "agent_eval_cases" ("organization_id", "business_unit_id", "agent_definition_id", "content_hash");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_definition"
    ON "agent_eval_cases" ("organization_id", "business_unit_id", "agent_definition_id", "status", "created_at" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_eval_cases_proposal"
    ON "agent_eval_cases" ("organization_id", "business_unit_id", "source_proposal_id")WHERE "source_proposal_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_thread"
    ON "agent_eval_cases" ("organization_id", "business_unit_id", "source_thread_id")WHERE "source_thread_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_expiry"
    ON "agent_eval_cases" ("expires_at")WHERE "expires_at" IS NOT NULL;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "eval_case_id" TEXT;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "checks" TEXT;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "judge" TEXT;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "case_score" REAL;

--bun:split

ALTER TABLE "agent_evaluations" ADD COLUMN "fingerprint" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_case"
    ON "agent_evaluations" ("organization_id", "business_unit_id", "eval_case_id", "created_at" DESC)WHERE "eval_case_id" IS NOT NULL;
