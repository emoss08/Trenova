--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "assistant_queued_messages"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "content" text NOT NULL,
    "request" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "position" bigint NOT NULL,
    "steer" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_assistant_queued_messages" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assistant_queued_messages_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_queued_messages_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_queued_messages_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_queued_messages_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_queued_messages_content" CHECK (length(btrim("content")) > 0 AND char_length("content") <= 16000),
    CONSTRAINT "ck_assistant_queued_messages_request" CHECK (jsonb_typeof("request") = 'object')
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assistant_queued_messages_thread_position" ON "assistant_queued_messages"("organization_id", "business_unit_id", "thread_id", "position");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assistant_queued_messages_user" ON "assistant_queued_messages"("user_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assistant_queued_messages_business_unit" ON "assistant_queued_messages"("business_unit_id");

--bun:split
COMMENT ON TABLE "assistant_queued_messages" IS 'Messages a person left for a conversation while its agent was working: sent in position order as each reply ends, or read into the running reply at its next step when steer is set';

--bun:split
COMMENT ON COLUMN "assistant_queued_messages"."steer" IS 'The person asked for the message to be read into the reply under way; one the reply ended before reading is sent as the next message';

--bun:split
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "world_changes" JSONB;

--bun:split
COMMENT ON COLUMN "assistant_messages"."world_changes" IS 'On a WorldChange notice: the records that changed elsewhere while the reply was written, as the reply was told';

--bun:split
ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split
ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction', 'Handoff', 'HandoffBrief', 'Steer', 'WorldChange'));

--bun:split
COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal; Delegated for a step another agent took on a task the conversation''s agent handed it; Schedule for a request the person scheduled, drawn as its schedule card; Compaction for the summary the model reads in place of the conversation before it; Handoff for the card left where a person handed the conversation to another agent; HandoffBrief for the summary that opens the handed-off conversation; Steer for what the person said while a reply was being written, read by it at its next step; WorldChange for the notice a reply was given when a record it was working with changed. Delegated, Schedule and Handoff are never replayed to the model';

--bun:split
SELECT trenova_rls.reconcile();
