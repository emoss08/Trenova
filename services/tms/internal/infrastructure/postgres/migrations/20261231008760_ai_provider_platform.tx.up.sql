-- An Anthropic Messages provider can reach Claude through a cloud platform as well as
-- at its base URL: Amazon Bedrock, Google Vertex AI, Claude Platform on AWS and on
-- Google Cloud, or Microsoft Foundry. Each signs in its own way and needs its own
-- region, project, workspace or resource, kept in platform_settings. None of those is
-- secret; the credential stays in api_key, encrypted. Every other kind is called
-- directly.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "platform" varchar(50) NOT NULL DEFAULT 'Direct',
    ADD COLUMN IF NOT EXISTS "platform_settings" jsonb;

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform" CHECK ("platform" IN ('Direct', 'Bedrock', 'VertexAI', 'ClaudePlatformAWS', 'ClaudePlatformGoogleCloud', 'Foundry'));

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform_kind" CHECK ("platform" = 'Direct' OR "kind" = 'AnthropicMessages');

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform_settings" CHECK (
        "platform_settings" IS NULL
        OR ("platform" <> 'Direct' AND jsonb_typeof("platform_settings") = 'object')
    );

--bun:split
COMMENT ON COLUMN "ai_providers"."platform" IS 'The service an Anthropic Messages provider reaches Claude through; Direct calls the base URL with the API key';

--bun:split
COMMENT ON COLUMN "ai_providers"."platform_settings" IS 'What the platform needs besides the credential: sign-in method, region, project, workspace, access key ID, resource, Entra tenant and client. Never secret';
