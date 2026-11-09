--keel:notx
-- Needs you reads every open item about the records on its first page in one statement.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_watchtower_items_open_about" ON "watchtower_items"("organization_id", "business_unit_id", "about_id") WHERE "resolved_at" IS NULL AND "about_id" IS NOT NULL;

--keel:split
-- Nova handled reads the newest settled writes across every tool: ran, failed or previewed.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_settled" ON "agent_proposals"("organization_id", "business_unit_id", (COALESCE("executed_at", "simulated_at")) DESC, "id" DESC) WHERE "executed_at" IS NOT NULL OR "simulated_at" IS NOT NULL;
