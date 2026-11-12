-- What a queued subject's work needs beyond its id: the event kind an agent
-- event carries, or the agent a routine runs for. The latest request wins.
ALTER TABLE "work_queue_items"
    ADD COLUMN IF NOT EXISTS "detail" varchar(200);

ALTER TABLE "work_queue_items"
    ALTER COLUMN "kind" TYPE varchar(120);

DROP INDEX IF EXISTS "idx_agent_runs_awaiting_decision";
