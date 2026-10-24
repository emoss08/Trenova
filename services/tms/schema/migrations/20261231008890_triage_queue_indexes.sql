DROP INDEX IF EXISTS "idx_inbound_messages_waiting_due";

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_waiting_queue" ON "inbound_messages"("organization_id", "business_unit_id", "due_at" ASC NULLS LAST, "received_at" DESC, "id" DESC) WHERE "status" IN ('InReview', 'Quarantined') AND "parked" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_census" ON "inbound_messages"("organization_id", "business_unit_id", "status", "classification", "mailbox_id");
