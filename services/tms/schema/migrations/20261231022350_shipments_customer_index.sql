--keel:notx
-- Shipment searches narrow by customer (a customer named in the words, a customer filter) and
-- page newest first, so the customer's shipments are read in that order from one index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_customer_created" ON "shipments"("organization_id", "business_unit_id", "customer_id", "created_at" DESC);
