--keel:notx
-- The receipt batch source picker reads a tenant's distinct sources in order, a page at a time.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_bank_receipt_import_batches_source" ON "bank_receipt_import_batches"("organization_id", "business_unit_id", "source");
