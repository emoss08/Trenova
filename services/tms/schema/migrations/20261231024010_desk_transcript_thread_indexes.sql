--keel:notx
-- The administrator's list of everyone's Desk conversations, and its export, read a tenant's
-- conversations newest last message first, keyset-paged on (last_message_at, id), narrowed to one
-- person, one agent, both or neither. Each shape has an index that leads with the tenant and ends in
-- the page's order, so a page reads its rows off the index whatever the tenant holds. The person's
-- index replaces idx_assistant_threads_user, whose columns it starts with.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_tenant_last_message" ON "assistant_threads"("organization_id", "business_unit_id", "last_message_at" DESC, "id" DESC);
--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_user_last_message" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "last_message_at" DESC, "id" DESC);
--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_threads_agent_last_message" ON "assistant_threads"("organization_id", "business_unit_id", "agent_definition_id", "last_message_at" DESC, "id" DESC);
--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_assistant_threads_user";
