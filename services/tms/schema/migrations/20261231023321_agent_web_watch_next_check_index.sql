--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_web_watches_next_check" ON "agent_web_watches"("next_check_at", "id")
WHERE
    "next_check_at" IS NOT NULL;
