DROP INDEX IF EXISTS "idx_assistant_messages_delegate_call";

--bun:split

DELETE FROM "assistant_messages" WHERE "kind" = 'Delegated';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote'));

--bun:split

ALTER TABLE "assistant_messages" DROP COLUMN IF EXISTS "delegate_call_id";

ALTER TABLE "assistant_messages" DROP COLUMN IF EXISTS "agent_definition_id";

--bun:split

ALTER TABLE "agent_definitions" DROP COLUMN IF EXISTS "delegate_ids";
