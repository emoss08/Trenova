CREATE INDEX IF NOT EXISTS "idx_inbound_messages_due" ON "inbound_messages"("organization_id", "business_unit_id", "due_at" ASC NULLS LAST, "received_at" DESC, "id" DESC);
