-- SQLite twin of 20261231008830_triage_customer_match.tx.up.sql.

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_customer" ON "inbound_messages"("organization_id", "business_unit_id", "matched_customer_id", "received_at") WHERE "matched_customer_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_replies_sent_to" ON "inbound_replies"("organization_id", "business_unit_id", "to_address", "sent_at") WHERE "status" = 'Sent';
