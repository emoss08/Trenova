DROP INDEX IF EXISTS "idx_inbound_messages_census";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_waiting_queue";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_waiting_due" ON "inbound_messages"("organization_id", "business_unit_id", "status", "due_at", "id") WHERE "parked" IS NULL;
