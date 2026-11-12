--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_plans_target" ON "agent_plans"("organization_id", "business_unit_id", "target_id", "created_at" DESC)
WHERE
    "target_id" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_solo_target" ON "agent_proposals"("organization_id", "business_unit_id", "target_id", "created_at" DESC)
WHERE
    "target_id" IS NOT NULL AND "plan_id" IS NULL;
