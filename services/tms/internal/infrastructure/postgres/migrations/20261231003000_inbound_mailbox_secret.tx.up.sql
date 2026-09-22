ALTER TABLE "inbound_mailboxes"
    ADD COLUMN IF NOT EXISTS "signing_secret" TEXT;

--bun:split

COMMENT ON COLUMN "inbound_mailboxes"."signing_secret" IS
    'The provider endpoint''s signing secret, encrypted at rest. It belongs to the mailbox rather than to the outbound email integration because they are separately rotated endpoints.';
