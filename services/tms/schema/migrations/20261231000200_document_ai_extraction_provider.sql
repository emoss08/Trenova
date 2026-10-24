-- Document extraction no longer goes straight to one vendor: the completion
-- router picks whichever configured provider serves the task.
--
-- A background handle is only meaningful to the endpoint that issued it, so the
-- provider has to be stored next to the response id. Polling the wrong endpoint
-- with someone else's handle is at best a 404 and at worst a read of another
-- organization's call, which is why the poll refuses a row with no provider
-- rather than guessing at one.
--
-- Rows written before this column existed keep a NULL provider. They were all
-- issued by the retiring OpenAI integration, and there is no honest value to
-- backfill: the ai_providers row they would name may not exist. Those polls
-- fail with a message telling the user to run the extraction again, which is
-- the truthful outcome and reaches at most the handful of extractions still in
-- flight when this deploys.

ALTER TABLE "document_ai_extractions"
    ADD COLUMN IF NOT EXISTS "provider_id" VARCHAR(100);
