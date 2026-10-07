CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_audit_entries_resource_time" ON "audit_entries"("organization_id", "business_unit_id", "resource_id", "timestamp");
