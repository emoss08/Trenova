-- SQLite twin of 20261231008920_reply_attachments.tx.up.sql.

ALTER TABLE "inbound_replies" ADD COLUMN "attachments" TEXT;

--bun:split
ALTER TABLE "email_message_attachments" ADD COLUMN "seal" TEXT CHECK ("seal" IS NULL OR "seal" IN ('Document'));
