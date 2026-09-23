-- An evaluation case is one question an agent was asked, frozen with what it was
-- given and what a good answer does, so the agent can be replayed against it
-- after every change and scored.
CREATE TABLE IF NOT EXISTS "agent_eval_cases" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "title" varchar(200),
    "source" varchar(30) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Candidate',
    "trigger" varchar(20) NOT NULL,
    "source_run_id" varchar(100),
    "source_turn_id" varchar(100),
    "source_thread_id" varchar(100),
    "source_message_id" varchar(100),
    "source_proposal_id" varchar(100),
    "source_feedback_id" varchar(100),
    "input" text NOT NULL,
    "history" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "page_context" jsonb,
    "mentions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "subject_type" varchar(50),
    "subject_id" varchar(100),
    "held_tools" text[] NOT NULL DEFAULT '{}',
    "tool_fixtures" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "expected" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "rubric" text,
    "redaction" jsonb,
    "content_hash" varchar(64) NOT NULL,
    "captured_fingerprint" jsonb,
    "expires_at" bigint,
    "created_by_user_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_agent_eval_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_eval_cases_source" CHECK ("source" IN ('DecidedProposal', 'ThumbsUp', 'Curated')),
    CONSTRAINT "ck_agent_eval_cases_status" CHECK ("status" IN ('Candidate', 'Active', 'Quarantined', 'Retired')),
    CONSTRAINT "ck_agent_eval_cases_subject" CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL)),
    CONSTRAINT "fk_agent_eval_cases_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_eval_cases_created_by" FOREIGN KEY ("created_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_eval_cases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_eval_cases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_eval_cases_content"
    ON "agent_eval_cases"("organization_id", "business_unit_id", "agent_definition_id", "content_hash");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_definition"
    ON "agent_eval_cases"("organization_id", "business_unit_id", "agent_definition_id", "status", "created_at" DESC);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_eval_cases_proposal"
    ON "agent_eval_cases"("organization_id", "business_unit_id", "source_proposal_id")
    WHERE "source_proposal_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_thread"
    ON "agent_eval_cases"("organization_id", "business_unit_id", "source_thread_id")
    WHERE "source_thread_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_eval_cases_expiry"
    ON "agent_eval_cases"("expires_at")
    WHERE "expires_at" IS NOT NULL;

COMMENT ON TABLE "agent_eval_cases" IS 'A question an agent was asked, frozen with what it was given and what a good answer does, replayed against the agent after every change and scored';

COMMENT ON COLUMN "agent_eval_cases"."agent_definition_id" IS 'The agent the case evaluates';

COMMENT ON COLUMN "agent_eval_cases"."title" IS 'A short name an administrator gives the case';

COMMENT ON COLUMN "agent_eval_cases"."source" IS 'Where the case came from: DecidedProposal for a proposal a person approved or rejected, ThumbsUp for a reply a person liked, Curated for one an administrator wrote';

COMMENT ON COLUMN "agent_eval_cases"."status" IS 'Candidate until an administrator activates it; Active cases run in every suite; Quarantined cases are kept but not run; Retired cases are kept for their history';

COMMENT ON COLUMN "agent_eval_cases"."trigger" IS 'How the original run started, which decides whether the case replays as a conversation or as a background run';

COMMENT ON COLUMN "agent_eval_cases"."source_run_id" IS 'The agent run the case was captured from, when there was one';

COMMENT ON COLUMN "agent_eval_cases"."source_turn_id" IS 'The assistant turn the case was captured from, when known';

COMMENT ON COLUMN "agent_eval_cases"."source_thread_id" IS 'The conversation the case was captured from; the case is purged when the conversation is deleted';

COMMENT ON COLUMN "agent_eval_cases"."source_message_id" IS 'The message the case was captured from';

COMMENT ON COLUMN "agent_eval_cases"."source_proposal_id" IS 'The decided proposal the case was captured from; at most one case per proposal';

COMMENT ON COLUMN "agent_eval_cases"."source_feedback_id" IS 'The feedback that marked the reply as good, when the case came from one';

COMMENT ON COLUMN "agent_eval_cases"."input" IS 'The question the agent is asked, as the person asked it';

COMMENT ON COLUMN "agent_eval_cases"."history" IS 'The conversation before the question, redacted when captured';

COMMENT ON COLUMN "agent_eval_cases"."page_context" IS 'What the person was looking at when they asked';

COMMENT ON COLUMN "agent_eval_cases"."mentions" IS 'The records the person named while asking';

COMMENT ON COLUMN "agent_eval_cases"."subject_type" IS 'The kind of record a background run concerned';

COMMENT ON COLUMN "agent_eval_cases"."subject_id" IS 'The record a background run concerned';

COMMENT ON COLUMN "agent_eval_cases"."held_tools" IS 'The tools the agent held when the case was captured; calling any other fails the case';

COMMENT ON COLUMN "agent_eval_cases"."tool_fixtures" IS 'Each tool call the original made: tool, arguments, the result with restricted fields redacted, and whether it failed';

COMMENT ON COLUMN "agent_eval_cases"."expected" IS 'What a good answer does: tools with per-argument tolerance rules, forbidden tools, proposals with the parameters a person approved or marked rejected, whether a refusal is expected, and phrases the reply must or must not mention';

COMMENT ON COLUMN "agent_eval_cases"."rubric" IS 'Guidance for a judge model scoring the reply';

COMMENT ON COLUMN "agent_eval_cases"."redaction" IS 'Which fields were replaced when the case was captured, and when';

COMMENT ON COLUMN "agent_eval_cases"."content_hash" IS 'SHA-256 of the frozen input; one case per agent per hash';

COMMENT ON COLUMN "agent_eval_cases"."captured_fingerprint" IS 'The agent version, prompt version, instructions hash and tools when the case was captured';

COMMENT ON COLUMN "agent_eval_cases"."expires_at" IS 'When the case is purged regardless of the retention period';

COMMENT ON COLUMN "agent_eval_cases"."created_by_user_id" IS 'The administrator who wrote or captured the case; empty for automatic capture';

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "source_run_id" DROP NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "subject_type" DROP NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations"
    ALTER COLUMN "subject_id" DROP NOT NULL;

--bun:split
ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "eval_case_id" varchar(100);

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "checks" jsonb;

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "judge" jsonb;

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "case_score" double precision;

ALTER TABLE "agent_evaluations"
    ADD COLUMN IF NOT EXISTS "fingerprint" jsonb;

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "fk_agent_evaluations_eval_case";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "fk_agent_evaluations_eval_case" FOREIGN KEY ("eval_case_id", "business_unit_id", "organization_id") REFERENCES "agent_eval_cases"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE;

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_source";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "ck_agent_evaluations_source" CHECK (num_nonnulls("source_run_id", "eval_case_id") = 1);

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "ck_agent_evaluations_case_score";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "ck_agent_evaluations_case_score" CHECK ("case_score" IS NULL OR ("case_score" >= 0 AND "case_score" <= 1));

--bun:split
ALTER TABLE "agent_evaluations"
    DROP CONSTRAINT IF EXISTS "chk_agent_evaluations_status";

--bun:split
ALTER TABLE "agent_evaluations"
    ADD CONSTRAINT "chk_agent_evaluations_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Failed', 'Skipped'));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_case"
    ON "agent_evaluations"("organization_id", "business_unit_id", "eval_case_id", "created_at" DESC)
    WHERE "eval_case_id" IS NOT NULL;

COMMENT ON COLUMN "agent_evaluations"."source_run_id" IS 'The recorded run replayed; empty when the evaluation replays an evaluation case';

COMMENT ON COLUMN "agent_evaluations"."eval_case_id" IS 'The evaluation case replayed; empty when the evaluation replays a recorded run';

COMMENT ON COLUMN "agent_evaluations"."checks" IS 'Each hard and soft check the replay was scored on, with the tool calls it made and whether it refused';

COMMENT ON COLUMN "agent_evaluations"."judge" IS 'A judge model''s score and rationale, when one was asked';

COMMENT ON COLUMN "agent_evaluations"."case_score" IS 'The case score from 0 to 1: the weighted checks blended with the judge, zero on a hard failure';

COMMENT ON COLUMN "agent_evaluations"."fingerprint" IS 'The agent version, prompt version, instructions hash and tools the replay ran against';

COMMENT ON COLUMN "agent_evaluations"."status" IS 'Pending, Running, Completed, Failed, or Skipped when the replay could not run as the original did, such as a conversation whose person no longer has an active account';
