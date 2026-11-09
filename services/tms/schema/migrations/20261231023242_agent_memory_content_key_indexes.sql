--keel:notx
-- The lookup a save makes before it keeps a memory, and the backstop for two saves of the same
-- words racing each other: one active, clean memory per readers, subject, tool and words.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_memories_content_key" ON "agent_memories"("organization_id", "business_unit_id", "scope", "content_key") WHERE "status" IN ('Active', 'Suggested');

--keel:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_agent_memories_kept_content" ON "agent_memories"("organization_id", "business_unit_id", "scope", "agent_definition_id", "owner_user_id", "role_id", "subject_type", "subject_id", "tool_name", "content_key") NULLS NOT DISTINCT WHERE "status" = 'Active' AND (NOT "tainted" OR "reviewed_at" IS NOT NULL);
