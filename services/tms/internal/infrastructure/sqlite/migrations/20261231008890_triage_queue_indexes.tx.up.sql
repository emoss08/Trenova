-- SQLite twin of 20261231008890_triage_queue_indexes.tx.up.sql.

DROP INDEX IF EXISTS "idx_inbound_messages_waiting_due";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_waiting_queue" ON "inbound_messages"("organization_id", "business_unit_id", "due_at", "received_at" DESC, "id" DESC) WHERE "status" IN ('InReview', 'Quarantined') AND "parked" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_census" ON "inbound_messages"("organization_id", "business_unit_id", "status", "classification", "mailbox_id");
