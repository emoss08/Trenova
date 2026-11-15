--keel:notx
-- The pages reading sums one feature's successful calls by hour across the instance. The index
-- leads with the feature so the reading is a range on the hours it asks for; the index the AI
-- spend meter read is gone with that meter.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_feature_created_at"
    ON "ai_usage_records"("feature", "created_at") INCLUDE ("succeeded", "pages");
--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_ai_usage_records_created_at";
