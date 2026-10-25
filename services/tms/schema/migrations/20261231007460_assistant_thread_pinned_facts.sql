-- What the person pinned for the agents to keep in mind for the whole
-- conversation: an ordered list of short facts. Every turn carries them in its
-- system prompt, so trimming the history never drops one.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "pinned_facts" JSONB;
