ALTER TABLE "inbound_replies" ADD COLUMN IF NOT EXISTS "attachments" jsonb;

--bun:split
ALTER TABLE "email_message_attachments" ADD COLUMN IF NOT EXISTS "seal" varchar(32);

--bun:split
ALTER TABLE "email_message_attachments" DROP CONSTRAINT IF EXISTS "ck_email_message_attachments_seal";

--bun:split
ALTER TABLE "email_message_attachments" ADD CONSTRAINT "ck_email_message_attachments_seal" CHECK ("seal" IS NULL OR "seal" IN ('Document'));
