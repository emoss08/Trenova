-- SQLite twin of 20261231008940_triage_due_index.tx.up.sql.

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_due" ON "inbound_messages"("organization_id", "business_unit_id", "due_at", "received_at" DESC, "id" DESC);
