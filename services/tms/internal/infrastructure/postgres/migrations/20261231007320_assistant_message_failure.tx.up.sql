-- Why a reply did not finish, kept with the reply so a conversation read back
-- later shows the same card the reader saw: which models were asked and what
-- each said, or that the reply was stopped.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "failure" JSONB;

--bun:split
COMMENT ON COLUMN "assistant_messages"."failure" IS 'Why the reply did not finish: kind (no_model, interrupted, stopped, before_start) and the models asked';
