-- SQLite twin of 20261231008920_reply_attachments.tx.down.sql.

ALTER TABLE "email_message_attachments" DROP COLUMN "seal";

--bun:split
ALTER TABLE "inbound_replies" DROP COLUMN "attachments";
