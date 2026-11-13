--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_eval_cases_automatic_age"
    ON "agent_eval_cases"("created_at")
    WHERE "automatic" AND "status" = 'Active';
