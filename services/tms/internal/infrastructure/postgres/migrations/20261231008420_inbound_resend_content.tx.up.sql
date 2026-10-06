ALTER TABLE "inbound_mailboxes"
    ADD COLUMN IF NOT EXISTS "provider_api_key" TEXT;

--bun:split
ALTER TABLE "inbound_message_attachments"
    ADD COLUMN IF NOT EXISTS "provider_attachment_id" VARCHAR(255);

--bun:split
COMMENT ON COLUMN "inbound_mailboxes"."provider_api_key" IS
    'The provider API key the mailbox reads message content with, encrypted at rest. Resend posts only metadata to the webhook, so the body, headers and attachments are fetched with this key.';

--bun:split
COMMENT ON COLUMN "inbound_message_attachments"."provider_attachment_id" IS
    'The provider''s own id for the file, so content fetched after the webhook lands on the row the webhook created.';
