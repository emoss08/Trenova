DROP INDEX IF EXISTS "idx_edi_messages_as2_message_id";

--bun:split
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "as2_message_id";

--bun:split
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "as2_mic";
