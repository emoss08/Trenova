-- A compaction summary is a message of its own kind: the model reads it in
-- place of the stretch of conversation it stands in for, which the column
-- below names by the sequence of its last message.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "compaction" JSONB;

--bun:split

COMMENT ON COLUMN "assistant_messages"."compaction" IS 'On a compaction summary: whether it started on its own, how many messages it replaces, the sequence of the last one (through), the context use either side, and what was kept in full';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction'));

COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal; Delegated for a step another agent took on a task the conversation''s agent handed it; Schedule for a request the person scheduled, drawn as its schedule card; Compaction for the summary the model reads in place of the conversation before it. Delegated and Schedule are never replayed to the model';

--bun:split

-- Every turn's history read looks for the latest summary first; only the
-- summaries are indexed, newest first within a thread.
CREATE INDEX IF NOT EXISTS "idx_assistant_messages_compaction"
    ON "assistant_messages" ("organization_id", "business_unit_id", "thread_id", "sequence" DESC)
    WHERE "kind" = 'Compaction';

--bun:split

ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split

ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp', 'Scheduled', 'Compaction'));

COMMENT ON COLUMN "assistant_turns"."origin" IS 'Person for a question somebody asked; DecisionFollowUp for the turn in which the agent reports what came of a decided proposal; Scheduled for a run of a conversation schedule; Compaction for the conversation being summarized to free its context';

--bun:split

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "context_usage" JSONB;

--bun:split

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "auto_compact_off" BOOLEAN NOT NULL DEFAULT FALSE;

--bun:split

COMMENT ON COLUMN "assistant_threads"."context_usage" IS 'How full the model''s context window was after the last turn or compaction, in estimated tokens by part, with the window it was measured against';

--bun:split

COMMENT ON COLUMN "assistant_threads"."auto_compact_off" IS 'The conversation no longer compacts itself on nearing a full context: the person turned it off, or cancelled a compaction that started on its own';
