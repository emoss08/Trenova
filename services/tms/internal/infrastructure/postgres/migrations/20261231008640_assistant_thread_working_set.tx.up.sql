-- The records a conversation is about, newest first: its subject, what the
-- person mentioned, what the agent read and what its writes changed. Every
-- turn reads each of them again before the model is asked anything, so a
-- status or an ETA from an earlier turn never stands in for the current one.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "working_set" JSONB;

--bun:split

COMMENT ON COLUMN "assistant_threads"."working_set" IS 'The records the conversation is about, newest first: kind, id, label, how it came into the conversation (subject, mention, page, read, wrote) and when it was last touched. Each turn reads them again before the model is asked anything';
