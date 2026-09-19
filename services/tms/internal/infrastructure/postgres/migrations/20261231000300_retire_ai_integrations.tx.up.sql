-- AI credentials moved to ai_providers, which carries the endpoint, the
-- protocol, the model and the tasks a provider may serve. The OpenAI
-- integration row held a bare API key and nothing else, so it goes rather than
-- sitting in the marketplace offering to configure something no code consults.
--
-- Only OpenAI is deleted. The Go side also carried a Type("Anthropic")
-- constant, but integration_type was never given that value
-- (20250415120000_integrations.tx.up.sql and every ALTER TYPE since), so no row
-- could ever hold it and naming it here only made PostgreSQL reject the
-- statement while coercing the literal.
--
-- The enum value stays: PostgreSQL cannot drop one, and nothing can write it
-- once the Go constant is gone.
DELETE FROM "integrations"
WHERE "type" = 'OpenAI';
