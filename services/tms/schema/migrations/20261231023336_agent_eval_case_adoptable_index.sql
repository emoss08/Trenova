--keel:notx
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_eval_cases_adoptable"
    ON "agent_eval_cases"("created_at")
    WHERE "status" = 'Candidate' AND "source" = 'DecidedProposal' AND "created_by_user_id" IS NULL;
