--keel:notx
-- A TA1 that rejects an interchange names it by its control number alone, and every outbound
-- message Trenova wrote in that interchange takes the rejection. The index finds them in one
-- range scan per partner instead of reading every message the partner was sent.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_edi_messages_interchange_ack" ON "edi_messages"("organization_id", "business_unit_id", "edi_partner_id", "direction", "interchange_control_number");
