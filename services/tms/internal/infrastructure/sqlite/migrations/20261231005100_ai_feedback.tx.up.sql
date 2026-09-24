-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005100_ai_feedback.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_feedback" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "target_type" TEXT NOT NULL,
    "target_id" TEXT NOT NULL,
    "target_part" TEXT NOT NULL DEFAULT '',
    "thread_id" TEXT,
    "turn_id" TEXT,
    "run_id" TEXT,
    "agent_definition_id" TEXT,
    "definition_version" INTEGER,
    "detector_key" TEXT,
    "task" TEXT,
    "model" TEXT,
    "provider_id" TEXT,
    "prompt_hash" TEXT,
    "tool_spec_hash" TEXT,
    "fingerprint_source" TEXT NOT NULL,
    "rating" INTEGER NOT NULL,
    "reasons" TEXT NOT NULL DEFAULT '[]',
    "comment" TEXT,
    "turn_snapshot" TEXT,
    "pattern_key" TEXT NOT NULL,
    "eval_case_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_feedback" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_ai_feedback_target" UNIQUE ("organization_id", "business_unit_id", "user_id", "target_type", "target_id", "target_part"),
    CONSTRAINT "ck_ai_feedback_rating" CHECK ("rating" IN (-1, 1)),
    CONSTRAINT "ck_ai_feedback_target_type" CHECK ("target_type" IN ('AssistantMessage', 'DelegatedAnswer', 'Briefing', 'BriefingSection', 'Insight', 'WatchtowerItem')),
    CONSTRAINT "ck_ai_feedback_fingerprint_source" CHECK ("fingerprint_source" IN ('AtTurn', 'AtRating', 'None')),
    CONSTRAINT "ck_ai_feedback_comment_length" CHECK ("comment" IS NULL OR length("comment") <= 1000),
    CONSTRAINT "ck_ai_feedback_pattern_key" CHECK (length("pattern_key") = 64),
    CONSTRAINT "fk_ai_feedback_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_feedback_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_feedback_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_feedback_agent_created"
    ON "ai_feedback" ("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC)WHERE "agent_definition_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_feedback_negative_pattern"
    ON "ai_feedback" ("organization_id", "business_unit_id", "agent_definition_id", "pattern_key", "created_at")WHERE "rating" = -1;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_feedback_created"
    ON "ai_feedback" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_feedback_thread"
    ON "ai_feedback" ("organization_id", "business_unit_id", "thread_id")WHERE "thread_id" IS NOT NULL;
