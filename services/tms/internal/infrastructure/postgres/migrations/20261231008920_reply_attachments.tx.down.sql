ALTER TABLE "email_message_attachments" DROP CONSTRAINT IF EXISTS "ck_email_message_attachments_seal";

--bun:split
ALTER TABLE "email_message_attachments" DROP COLUMN IF EXISTS "seal";

--bun:split
ALTER TABLE "inbound_replies" DROP COLUMN IF EXISTS "attachments";
