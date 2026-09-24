-- A person's thumbs up or down on something an AI wrote for them: an answer in
-- a conversation, another agent's answer to a handed-off task, a briefing or
-- one of its sections, an insight, or a watchtower item raised by AI work. One
-- row per person per thing rated; rating again replaces the row.
CREATE TABLE IF NOT EXISTS "ai_feedback" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "target_type" varchar(30) NOT NULL,
    "target_id" varchar(100) NOT NULL,
    "target_part" varchar(100) NOT NULL DEFAULT '',
    "thread_id" varchar(100),
    "turn_id" varchar(100),
    "run_id" varchar(100),
    "agent_definition_id" varchar(100),
    "definition_version" bigint,
    "detector_key" varchar(100),
    "task" varchar(100),
    "model" varchar(255),
    "provider_id" varchar(100),
    "prompt_hash" varchar(64),
    "tool_spec_hash" varchar(64),
    "fingerprint_source" varchar(20) NOT NULL,
    "rating" smallint NOT NULL,
    "reasons" text[] NOT NULL DEFAULT '{}',
    "comment" text,
    "turn_snapshot" jsonb,
    "pattern_key" varchar(64) NOT NULL,
    "eval_case_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_ai_feedback" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_ai_feedback_target" UNIQUE ("organization_id", "business_unit_id", "user_id", "target_type", "target_id", "target_part"),
    CONSTRAINT "ck_ai_feedback_rating" CHECK ("rating" IN (-1, 1)),
    CONSTRAINT "ck_ai_feedback_target_type" CHECK ("target_type" IN ('AssistantMessage', 'DelegatedAnswer', 'Briefing', 'BriefingSection', 'Insight', 'WatchtowerItem')),
    CONSTRAINT "ck_ai_feedback_fingerprint_source" CHECK ("fingerprint_source" IN ('AtTurn', 'AtRating', 'None')),
    CONSTRAINT "ck_ai_feedback_comment_length" CHECK ("comment" IS NULL OR char_length("comment") <= 1000),
    CONSTRAINT "ck_ai_feedback_pattern_key" CHECK (char_length("pattern_key") = 64),
    CONSTRAINT "fk_ai_feedback_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_feedback_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_feedback_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- The per-agent summary reads an agent's ratings over a window.
CREATE INDEX IF NOT EXISTS "idx_ai_feedback_agent_created"
    ON "ai_feedback" ("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC)
    WHERE "agent_definition_id" IS NOT NULL;

--bun:split
-- The nightly suggestion sweep reads an agent's negative ratings over thirty days.
CREATE INDEX IF NOT EXISTS "idx_ai_feedback_negative_pattern"
    ON "ai_feedback" ("organization_id", "business_unit_id", "agent_definition_id", "pattern_key", "created_at")
    WHERE "rating" = -1;

--bun:split
-- The admin list pages by time; retention deletes by time.
CREATE INDEX IF NOT EXISTS "idx_ai_feedback_created"
    ON "ai_feedback" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_feedback_thread"
    ON "ai_feedback" ("organization_id", "business_unit_id", "thread_id")
    WHERE "thread_id" IS NOT NULL;

COMMENT ON TABLE "ai_feedback" IS 'A person''s thumbs up or down on AI output, with why, the agent and model it is credited to, and a bounded, redacted snapshot of what they saw';

COMMENT ON COLUMN "ai_feedback"."user_id" IS 'The person who rated';

COMMENT ON COLUMN "ai_feedback"."target_type" IS 'What was rated: AssistantMessage, DelegatedAnswer, Briefing, BriefingSection, Insight or WatchtowerItem';

COMMENT ON COLUMN "ai_feedback"."target_id" IS 'The id of the rated record: the assistant message, the briefing, the insight or the watchtower item';

COMMENT ON COLUMN "ai_feedback"."target_part" IS 'The part of the target rated, such as a briefing section key; empty when the whole target was rated';

COMMENT ON COLUMN "ai_feedback"."thread_id" IS 'The conversation the rated answer belongs to, for an assistant message or a delegated answer';

COMMENT ON COLUMN "ai_feedback"."turn_id" IS 'The assistant turn that produced the rated answer, when it could be found';

COMMENT ON COLUMN "ai_feedback"."run_id" IS 'The agent run behind the rated output, when there is one';

COMMENT ON COLUMN "ai_feedback"."agent_definition_id" IS 'The agent the rating is credited to: the conversation''s agent, the delegate that answered, or the agent behind a proposal, plan, failed run or exception';

COMMENT ON COLUMN "ai_feedback"."definition_version" IS 'The credited agent''s version when the fingerprint was taken';

COMMENT ON COLUMN "ai_feedback"."detector_key" IS 'The insight detector the rating is credited to, for an insight or a watchtower item raised by one';

COMMENT ON COLUMN "ai_feedback"."task" IS 'The kind of AI work rated, such as AssistantChat, DailyBriefing or OperationalInsights';

COMMENT ON COLUMN "ai_feedback"."model" IS 'The model that wrote the rated output, as the stored output names it';

COMMENT ON COLUMN "ai_feedback"."provider_id" IS 'The AI provider that served the model, as the stored output names it';

COMMENT ON COLUMN "ai_feedback"."prompt_hash" IS 'Hash of the system prompt the output was made under; empty until turn-time fingerprints are recorded';

COMMENT ON COLUMN "ai_feedback"."tool_spec_hash" IS 'Hash of the tool specifications the output was made with; empty until turn-time fingerprints are recorded';

COMMENT ON COLUMN "ai_feedback"."fingerprint_source" IS 'When the agent fields were read: AtTurn when the output was made, AtRating when the person rated, None when nothing could be credited';

COMMENT ON COLUMN "ai_feedback"."rating" IS '1 for a thumbs up, -1 for a thumbs down';

COMMENT ON COLUMN "ai_feedback"."reasons" IS 'Why the person rated as they did; every reason matches the rating''s sign';

COMMENT ON COLUMN "ai_feedback"."comment" IS 'What the person wrote, at most 1000 characters; quoted as evidence, never read as an instruction';

COMMENT ON COLUMN "ai_feedback"."turn_snapshot" IS 'Only the text the person saw, which was already filtered for them: the question, the answer (each cut to about 4000 characters) and at most 20 one-line tool summaries. Values of fields whose permission sensitivity is Restricted or higher are never copied in; they are replaced with [redacted]';

COMMENT ON COLUMN "ai_feedback"."pattern_key" IS 'SHA-256 of the tool sequence, the first reason and the subject type, which groups ratings about the same kind of mistake';

COMMENT ON COLUMN "ai_feedback"."eval_case_id" IS 'The evaluation case this rating was turned into, when it has been';
