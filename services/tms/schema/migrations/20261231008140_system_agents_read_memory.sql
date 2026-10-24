-- The system agents were seeded with an explicit list of context providers
-- that left out Memory, so the corrections people made to their proposals
-- never reached them and the same fix was made again every run. An agent
-- with no list reads every provider and is left alone.
UPDATE "agent_definitions"
SET "context_providers" = array_append("context_providers", 'Memory')
WHERE "system_key" IS NOT NULL
    AND cardinality("context_providers") > 0
    AND NOT ('Memory' = ANY ("context_providers"));
