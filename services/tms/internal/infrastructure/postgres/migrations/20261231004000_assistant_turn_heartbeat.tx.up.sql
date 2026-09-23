-- A turn running inside an API process has nothing outside it that notices
-- when the process dies: the record stays Running, the conversation's one live
-- slot stays held, and every later question waits on a reply that is never
-- coming. The running turn now writes when it was last alive, so a turn that
-- stopped writing can be told apart from one that is only thinking.
ALTER TABLE "assistant_turns"
    ADD COLUMN IF NOT EXISTS "heartbeat_at" BIGINT;

--bun:split

UPDATE "assistant_turns"
SET "heartbeat_at" = COALESCE("started_at", "created_at")
WHERE "heartbeat_at" IS NULL;

--bun:split

COMMENT ON COLUMN "assistant_turns"."heartbeat_at" IS 'When a turn running in an API process last proved it was alive; a live in-process turn whose heartbeat is stale died with its process';
