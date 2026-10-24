--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_tenant_updated" ON "agent_runs"("organization_id", "business_unit_id", "updated_at");

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_turns_tenant_updated" ON "assistant_turns"("organization_id", "business_unit_id", "updated_at");
