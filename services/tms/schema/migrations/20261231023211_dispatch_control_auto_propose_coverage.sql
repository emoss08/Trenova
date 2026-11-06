-- Whether an organization keeps a proposed driver waiting on every uncovered
-- move. Off until someone turns it on: it plans on its own, and each plan is a
-- decision waiting on a dispatcher.
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "auto_propose_coverage" boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN "dispatch_controls"."auto_propose_coverage" IS 'Keep a proposed driver waiting on every uncovered move, re-planned when loads are created or uncovered and every hour';
