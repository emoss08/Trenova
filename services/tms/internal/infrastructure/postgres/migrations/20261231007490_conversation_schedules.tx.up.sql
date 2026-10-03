-- A conversation schedule is a request somebody asked the Desk to repeat:
-- "every weekday at 7:30, what's blocking the billing queue?". Each one is a
-- Temporal Schedule named conversation-schedule/{id}; this row is what the
-- schedule is made from and what the person sees, and the answers land as
-- scheduled turns in the conversation the request was made in.
CREATE TABLE IF NOT EXISTS "conversation_schedules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    -- user_id owns the schedule. Every run is a turn in their conversation,
    -- asked with their access, as if they had typed it.
    "user_id" varchar(100) NOT NULL,
    "prompt" text NOT NULL,
    "cadence" varchar(100) NOT NULL,
    "cron_expression" varchar(100) NOT NULL,
    "timezone" varchar(100) NOT NULL DEFAULT 'UTC',
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "last_run_at" bigint,
    "next_run_at" bigint,
    "last_turn_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_conversation_schedules" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_conversation_schedules_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_conversation_schedules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_conversation_schedules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- A conversation's schedules, newest first: the cards in the thread.
CREATE INDEX IF NOT EXISTS "idx_conversation_schedules_thread" ON "conversation_schedules"("organization_id", "business_unit_id", "thread_id", "created_at" DESC);

--bun:split
-- One person's schedules across their conversations, newest first.
CREATE INDEX IF NOT EXISTS "idx_conversation_schedules_user" ON "conversation_schedules"("organization_id", "business_unit_id", "user_id", "created_at" DESC);

--bun:split
COMMENT ON COLUMN "conversation_schedules"."cadence" IS 'The cadence as the person reads it, such as Every weekday · 7:30 AM';

--bun:split
COMMENT ON COLUMN "conversation_schedules"."cron_expression" IS 'Five-field cron the Temporal schedule fires on, read in timezone';

--bun:split
COMMENT ON COLUMN "conversation_schedules"."last_turn_id" IS 'The assistant turn the latest run started';

--bun:split
-- The card a schedule is drawn as in its conversation is a message of its
-- own, so it is read back with the history and placed where it was asked.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "schedule_id" VARCHAR(100);

COMMENT ON COLUMN "assistant_messages"."schedule_id" IS 'The conversation schedule a Schedule message created; the schedule may since have been deleted';

--bun:split
ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split
ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule'));

COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal; Delegated for a step another agent took on a task the conversation''s agent handed it; Schedule for a request the person scheduled, drawn as its schedule card. Delegated and Schedule are never replayed to the model';

--bun:split
ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split
ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp', 'Scheduled'));

COMMENT ON COLUMN "assistant_turns"."origin" IS 'Person for a question somebody asked; DecisionFollowUp for the turn in which the agent reports what came of a decided proposal; Scheduled for a run of a conversation schedule';
