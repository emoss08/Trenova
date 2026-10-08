CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_provider_spend" ON "ai_usage_records"("organization_id", "business_unit_id", "provider_id", "created_at") WHERE "cost_usd" IS NOT NULL;
