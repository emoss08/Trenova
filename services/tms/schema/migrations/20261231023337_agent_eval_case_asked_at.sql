-- A case replayed against the reads its original was given runs as of the
-- moment the person asked, so a recorded result that says "tomorrow" and the
-- date the agent is told agree.
ALTER TABLE "agent_eval_cases"
    ADD COLUMN IF NOT EXISTS "asked_at" bigint;

COMMENT ON COLUMN "agent_eval_cases"."asked_at" IS 'When the person asked the question the case froze. A replay answered from recorded tool results runs with its clock at this moment; empty for a case written by hand or captured from a background run, which replays as of now';

COMMENT ON COLUMN "agent_eval_cases"."tool_fixtures" IS 'Each tool call the original made: tool, arguments, the result with restricted fields redacted, and whether it failed. A replay answers the same call with the recorded result, unless a field was redacted from it';
