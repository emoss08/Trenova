-- An Anthropic Messages provider can reach Claude through a cloud platform as well as
-- at its base URL: Amazon Bedrock, Google Vertex AI, Claude Platform on AWS and on
-- Google Cloud, or Microsoft Foundry. Each signs in its own way and needs its own
-- region, project, workspace or resource, kept in platform_settings. None of those is
-- secret; the credential stays in api_key, encrypted. Every other kind is called
-- directly. It first shipped as 20261231008760, a version master also gave the
-- assistant_threads touched index, so it is renumbered here and written to run again
-- on a database that applied it under the old number.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "platform" varchar(50) NOT NULL DEFAULT 'Direct',
    ADD COLUMN IF NOT EXISTS "platform_settings" jsonb;

ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform";

ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform" CHECK ("platform" IN ('Direct', 'Bedrock', 'VertexAI', 'ClaudePlatformAWS', 'ClaudePlatformGoogleCloud', 'Foundry'));

ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform_kind";

ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform_kind" CHECK ("platform" = 'Direct' OR "kind" = 'AnthropicMessages');

ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_platform_settings";

ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_platform_settings" CHECK (
        "platform_settings" IS NULL
        OR ("platform" <> 'Direct' AND jsonb_typeof("platform_settings") = 'object')
    );

COMMENT ON COLUMN "ai_providers"."platform" IS 'The service an Anthropic Messages provider reaches Claude through; Direct calls the base URL with the API key';

COMMENT ON COLUMN "ai_providers"."platform_settings" IS 'What the platform needs besides the credential: sign-in method, region, project, workspace, access key ID, resource, Entra tenant and client. Never secret';
