-- The row carried an encrypted API key that only the organization knew. Nothing
-- can recreate it, so rolling back leaves the marketplace entry absent and an
-- administrator re-enters the key against an AI provider instead.
SELECT 1;
