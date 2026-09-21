-- A pending proposal from a background run is told to the people who can
-- decide it when it is recorded. If it is still pending hours later, they are
-- told once more, with more weight and by email as well. This column is what
-- keeps that second notice to one: the sweep only picks proposals it has not
-- reminded anyone about.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "reminded_at" bigint;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_pending_unreminded"
    ON "agent_proposals"("created_at")
    WHERE "status" = 'Pending' AND "reminded_at" IS NULL;

COMMENT ON COLUMN "agent_proposals"."reminded_at" IS 'When the deciders were reminded that this proposal was still pending; null until they are';
