ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "as2_message_id" varchar(255);

--bun:split
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "as2_mic" varchar(255);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_messages_as2_message_id" ON "edi_messages"("as2_message_id")
WHERE
    "as2_message_id" IS NOT NULL;
