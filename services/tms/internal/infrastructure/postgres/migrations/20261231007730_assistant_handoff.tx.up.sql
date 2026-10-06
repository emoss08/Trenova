-- A conversation a person hands to another agent: the new conversation keeps
-- the one it came from, and both carry a message recording what went over.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "handed_from_thread_id" VARCHAR(100);

COMMENT ON COLUMN "assistant_threads"."handed_from_thread_id" IS 'The conversation this one was handed off from; null for one a person started';

--bun:split

ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "handoff" JSONB;

COMMENT ON COLUMN "assistant_messages"."handoff" IS 'What a hand-off carried: set on the Handoff card in the conversation handed off and on the HandoffBrief that opens the new one';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction', 'Handoff', 'HandoffBrief'));

COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal; Delegated for a step another agent took on a task the conversation''s agent handed it; Schedule for a request the person scheduled, drawn as its schedule card; Compaction for the summary the model reads in place of the conversation before it; Handoff for the card left where a person handed the conversation to another agent; HandoffBrief for the summary that opens the handed-off conversation. Delegated, Schedule and Handoff are never replayed to the model';
