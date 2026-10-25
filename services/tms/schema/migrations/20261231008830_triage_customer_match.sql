CREATE INDEX IF NOT EXISTS "idx_inbound_messages_customer" ON "inbound_messages"("organization_id", "business_unit_id", "matched_customer_id", "received_at" DESC) WHERE "matched_customer_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_sent_to" ON "inbound_replies"("organization_id", "business_unit_id", lower(split_part("to_address", '@', 2)), "sent_at" DESC) WHERE "status" = 'Sent';
