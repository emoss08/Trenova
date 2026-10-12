-- SQLite twin of 20261231008830_triage_customer_match.tx.down.sql.

DROP INDEX IF EXISTS "idx_inbound_replies_sent_to";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_customer";
