--keel:notx
-- Commodity search matches name and description with ILIKE, and a search that finds nothing falls
-- back to a close spelling of the name (word_similarity through the <% operator): both are served by
-- trigram indexes, as customers, locations and workers already are.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_commodities_trgm_name" ON "commodities" USING gin("name" gin_trgm_ops);

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_commodities_trgm_description" ON "commodities" USING gin("description" gin_trgm_ops);
