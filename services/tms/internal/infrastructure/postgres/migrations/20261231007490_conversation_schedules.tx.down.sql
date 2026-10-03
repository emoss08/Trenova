UPDATE "assistant_turns" SET "origin" = 'Person' WHERE "origin" = 'Scheduled';

--bun:split

ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split

ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp'));

--bun:split

DELETE FROM "assistant_messages" WHERE "kind" = 'Schedule';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated'));

--bun:split

ALTER TABLE "assistant_messages" DROP COLUMN IF EXISTS "schedule_id";

--bun:split

DROP TABLE IF EXISTS "conversation_schedules";
