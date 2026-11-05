--keel:notx
-- A managed instance reports its usage to the control plane from the source rows: the AI calls
-- of each hour across every organization, and the bytes of every stored document. Both read the
-- whole instance, so each gets an index that answers it without a tenant prefix.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_created_at"
    ON "ai_usage_records"("created_at") INCLUDE ("cost_usd", "feature", "succeeded");

CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_documents_file_size"
    ON "documents"("file_size");
