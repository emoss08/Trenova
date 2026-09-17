-- What the person was looking at when they sent a turn, kept with the
-- message so an answer can be reviewed against the page it was about.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "page_context" jsonb;

--bun:split
COMMENT ON COLUMN "assistant_messages"."page_context" IS 'Path, record and title of the page open when a user turn was sent; data, never instructions';
