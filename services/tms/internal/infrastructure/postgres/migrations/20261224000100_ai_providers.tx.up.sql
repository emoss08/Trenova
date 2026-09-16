-- An organization configures one row per model endpoint rather than one
-- credential per vendor, so it can run a local model for high-volume
-- classification alongside a hosted one for work that reaches the ledger.
CREATE TABLE IF NOT EXISTS "ai_providers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    -- Wire protocol, not vendor: one OpenAIChat row reaches vLLM, SGLang,
    -- OpenRouter, Groq, Together, Fireworks, Bedrock, or anything else speaking
    -- that shape. Ollama is separate only because its compat endpoint ignores
    -- json_schema while its native endpoint honours it.
    "kind" varchar(50) NOT NULL,
    "base_url" text,
    -- Free text: self-hosted model names are arbitrary and cannot be enumerated.
    "model" varchar(200) NOT NULL,
    "api_key" text,
    "allow_private_network" boolean NOT NULL DEFAULT FALSE,
    "structured_output_mode" varchar(50) NOT NULL,
    "max_tokens" integer NOT NULL DEFAULT 8192,
    "tasks" text[],
    "priority" integer NOT NULL DEFAULT 100,
    "trusted" boolean NOT NULL DEFAULT FALSE,
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_ai_providers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_providers_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_providers_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_providers_kind" CHECK ("kind" IN ('AnthropicMessages', 'OpenAIResponses', 'OpenAIChat', 'Ollama')),
    CONSTRAINT "ck_ai_providers_structured_output_mode" CHECK ("structured_output_mode" IN ('JSONSchema', 'JSONMode', 'Prompted')),
    CONSTRAINT "ck_ai_providers_max_tokens" CHECK ("max_tokens" BETWEEN 256 AND 200000),
    CONSTRAINT "ck_ai_providers_priority" CHECK ("priority" >= 0)
);

--bun:split
-- A name is how an operator tells two endpoints apart in the routing UI, so
-- duplicates within an organization would make the configuration ambiguous.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_providers_org_name" ON "ai_providers"("organization_id", "business_unit_id", lower("name"));

--bun:split
-- Routing reads enabled providers for a task in priority order; this index is
-- what keeps that resolution off a sequential scan on every AI call.
CREATE INDEX IF NOT EXISTS "idx_ai_providers_routing" ON "ai_providers"("organization_id", "business_unit_id", "enabled", "priority");

CREATE INDEX IF NOT EXISTS "idx_ai_providers_tasks" ON "ai_providers" USING GIN("tasks");

--bun:split
COMMENT ON TABLE "ai_providers" IS 'Per-organization AI model endpoints, including self-hosted OpenAI-compatible servers';

COMMENT ON COLUMN "ai_providers"."kind" IS 'Wire protocol spoken by the endpoint, not the vendor operating it';

COMMENT ON COLUMN "ai_providers"."api_key" IS 'Encrypted at rest by the application encryption service; never returned to clients';

COMMENT ON COLUMN "ai_providers"."allow_private_network" IS 'Permits egress to loopback and RFC 1918 addresses for a self-hosted endpoint; link-local stays blocked so cloud metadata remains unreachable';

COMMENT ON COLUMN "ai_providers"."trusted" IS 'Administrator vouches for this endpoint; required before it can serve a task that changes financial records';

COMMENT ON COLUMN "ai_providers"."structured_output_mode" IS 'How far the endpoint can be trusted to honour a JSON schema, which drives the parsing strategy';
