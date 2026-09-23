ALTER TABLE "assistant_messages" DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

ALTER TABLE "assistant_messages" DROP COLUMN IF EXISTS "kind";
