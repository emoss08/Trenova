ALTER TABLE "inbound_message_attachments"
    DROP COLUMN IF EXISTS "provider_attachment_id";

--bun:split
ALTER TABLE "inbound_mailboxes"
    DROP COLUMN IF EXISTS "provider_api_key";
