ALTER TABLE "agent_runs"
    ADD COLUMN IF NOT EXISTS "transcript" JSONB;

COMMENT ON COLUMN "agent_runs"."transcript" IS 'What the run''s model said and the tools it called, bounded: a message over 64 KiB keeps only who said it and what it called, and past 256 KiB the middle of the run is left out and counted. Null for a run filed before transcripts were kept or one that produced no messages. Deleted with its run';
