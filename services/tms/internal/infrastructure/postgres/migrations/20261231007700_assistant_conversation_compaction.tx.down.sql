ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "auto_compact_off";

--bun:split

ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "context_usage";

--bun:split

DELETE FROM "assistant_turns"
WHERE "origin" = 'Compaction';

--bun:split

ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split

ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp', 'Scheduled'));

--bun:split

DROP INDEX IF EXISTS "idx_assistant_messages_compaction";

--bun:split

DELETE FROM "assistant_messages"
WHERE "kind" = 'Compaction';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule'));

--bun:split

ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "compaction";
