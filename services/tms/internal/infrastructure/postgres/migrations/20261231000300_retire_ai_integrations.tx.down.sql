-- The rows carried encrypted API keys that only the organization knew. Nothing
-- can recreate them, so rolling back leaves the marketplace entries absent and
-- an administrator re-enters the key against an AI provider instead.
SELECT 1;
