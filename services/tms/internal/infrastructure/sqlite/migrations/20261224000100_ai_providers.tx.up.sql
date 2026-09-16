-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261224000100_ai_providers.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_providers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL,
    "base_url" TEXT,
    "model" TEXT NOT NULL,
    "api_key" TEXT,
    "allow_private_network" INTEGER NOT NULL DEFAULT 0,
    "structured_output_mode" TEXT NOT NULL,
    "max_tokens" INTEGER NOT NULL DEFAULT 8192,
    "tasks" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 100,
    "trusted" INTEGER NOT NULL DEFAULT 0,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_providers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_providers_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_providers_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_providers_kind" CHECK ("kind" IN ('AnthropicMessages', 'OpenAIResponses', 'OpenAIChat', 'Ollama')),
    CONSTRAINT "ck_ai_providers_structured_output_mode" CHECK ("structured_output_mode" IN ('JSONSchema', 'JSONMode', 'Prompted')),
    CONSTRAINT "ck_ai_providers_max_tokens" CHECK ("max_tokens" BETWEEN 256 AND 200000),
    CONSTRAINT "ck_ai_providers_priority" CHECK ("priority" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_providers_org_name" ON "ai_providers" ("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_providers_routing" ON "ai_providers" ("organization_id", "business_unit_id", "enabled", "priority");
