--keel:notx
-- A board's day of one trading partner's tenders reads that partner's inbound files received in
-- one day: an index range scan on the partner and the receipt time, not every file the tenant
-- ever received from it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_edi_inbound_files_partner_received" ON "edi_inbound_files"("organization_id", "business_unit_id", "edi_partner_id", "received_at");

--keel:split
-- The 990 that answered a tender, and the messages of one load tender (edi_list_messages'
-- transferId), are read by the transfer they belong to.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_edi_messages_transfer" ON "edi_messages"("transfer_id")
WHERE
    "transfer_id" IS NOT NULL;
