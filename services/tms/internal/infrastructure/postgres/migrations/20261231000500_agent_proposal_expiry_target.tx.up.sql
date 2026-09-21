-- A proposal is a judgement about the world as it was. Past its expiry that
-- world is gone and the judgement with it, so a pending proposal now carries
-- when it stops being decidable. Rows from before this column existed keep a
-- null expiry and never expire on their own; the sweeper only touches rows
-- whose expiry has passed.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "expires_at" bigint;

--bun:split
-- The record a proposal would change, pinned at the version it had when the
-- change was proposed. The executor refuses to run against a different version:
-- a hold proposed on a shipment that has since been delivered is not the change
-- anyone approved. Null for tools that act on no single record.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "target_resource" varchar(100);

--bun:split
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "target_id" varchar(100);

--bun:split
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "target_version" bigint;

--bun:split
-- The sweeper asks one question every quarter hour: which pending proposals
-- have expired. This is that question's index.
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_pending_expiry"
    ON "agent_proposals"("expires_at")
    WHERE "status" = 'Pending' AND "expires_at" IS NOT NULL;

COMMENT ON COLUMN "agent_proposals"."expires_at" IS 'When a pending proposal stops being decidable; null means it predates expiry and never expires on its own';

COMMENT ON COLUMN "agent_proposals"."target_resource" IS 'Permission resource of the one record the proposal would change, when the tool names one';

COMMENT ON COLUMN "agent_proposals"."target_id" IS 'Id of the record the proposal would change';

COMMENT ON COLUMN "agent_proposals"."target_version" IS 'Version that record had when the change was proposed; execution refuses a different one';
