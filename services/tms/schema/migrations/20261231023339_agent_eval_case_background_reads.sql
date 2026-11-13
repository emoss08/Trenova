-- A case captured from a run an agent started on its own now keeps what that
-- run's tools returned, read from the run's transcript, and when the run
-- started, the same as a case captured from a conversation.
COMMENT ON COLUMN "agent_eval_cases"."asked_at" IS 'When the person asked the question the case froze, or when the background run that made the proposal started. A replay answered from recorded tool results runs with its clock at this moment; empty for a case written by hand, which replays as of now';

COMMENT ON COLUMN "agent_eval_cases"."tool_fixtures" IS 'Each tool call the original made, in a conversation turn or a background run: tool, arguments, the result with restricted fields redacted, and whether it failed. A replay answers the same call with the recorded result, unless a field was redacted from it';
