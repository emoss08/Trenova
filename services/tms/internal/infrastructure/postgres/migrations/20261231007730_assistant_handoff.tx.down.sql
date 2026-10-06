DELETE FROM "assistant_messages" WHERE "kind" IN ('Handoff', 'HandoffBrief');

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction'));

ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "handoff";

ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "handed_from_thread_id";
