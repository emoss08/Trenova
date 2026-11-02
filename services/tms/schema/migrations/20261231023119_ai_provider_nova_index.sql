--keel:notx
-- An organization runs Nova on one provider, and every Nova call reads it by this index.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_providers_nova" ON "ai_providers"("organization_id", "business_unit_id")
WHERE
    "nova";
