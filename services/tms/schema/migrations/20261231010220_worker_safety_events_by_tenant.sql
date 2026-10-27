-- Safety events are read across the organization, newest first: every worker's
-- record, or only what is still open or under review. The worker index serves
-- one worker's record; these serve the organization's without sorting every
-- event it ever recorded.
CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_tenant" ON "worker_safety_events"("organization_id", "business_unit_id", "occurred_at" DESC, "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_worker_safety_events_open" ON "worker_safety_events"("organization_id", "business_unit_id", "occurred_at" DESC, "created_at" DESC)
WHERE
    "status" IN ('Open', 'UnderReview');
