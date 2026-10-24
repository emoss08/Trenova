-- An approved proposal used to keep only that it ran, so the conversation
-- that raised it knew a report had been saved but not which one, and the
-- agent passed the proposal's own id where the report's was wanted. This
-- column keeps what the run made: the record's kind, name and ids, bounded
-- by the application to about a kilobyte. Never a tool's full output.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "execution_result" jsonb;

COMMENT ON COLUMN "agent_proposals"."execution_result" IS 'What an executed proposal made (kind, name and ids of the record), when the tool reports it; null otherwise';
