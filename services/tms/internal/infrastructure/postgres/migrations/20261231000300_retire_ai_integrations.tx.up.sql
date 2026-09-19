-- AI credentials moved to ai_providers, which carries the endpoint, the
-- protocol, the model and the tasks a provider may serve. The two integration
-- rows that used to hold a bare API key have nothing left to configure, so they
-- go rather than sitting in the marketplace doing nothing.
--
-- The enum values stay: PostgreSQL cannot drop one, and nothing can write them
-- once the Go constants are gone.
DELETE FROM "integrations"
WHERE "type" IN ('OpenAI', 'Anthropic');
