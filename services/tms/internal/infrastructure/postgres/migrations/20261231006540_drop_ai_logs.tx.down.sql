-- Restores the table as it stood before it was dropped, empty: the rows it held
-- were not kept anywhere.
CREATE TYPE "operation_enum" AS ENUM(
    'ClassifyLocation',
    'DocumentIntelligenceRoute',
    'DocumentIntelligenceExtract',
    'ShipmentImportChat',
    'FormulaGenerate',
    'FormulaExplain'
);

--bun:split
CREATE TABLE IF NOT EXISTS "ai_logs"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "prompt" text NOT NULL,
    "response" text NOT NULL,
    "model" varchar(200) NOT NULL,
    "operation" operation_enum NOT NULL,
    "object" varchar(100) NOT NULL,
    "service_tier" varchar(100) NOT NULL,
    "prompt_tokens" integer NOT NULL,
    "completion_tokens" integer NOT NULL,
    "total_tokens" integer NOT NULL,
    "reasoning_tokens" integer NOT NULL,
    "timestamp" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "provider_kind" varchar(50) NOT NULL DEFAULT '',
    "provider_id" varchar(100),
    CONSTRAINT "pk_ai_logs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_logs_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("prompt", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("operation"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("response", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("model", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_logs_timestamp" ON "ai_logs"("timestamp");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_org_bu" ON "ai_logs"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_user" ON "ai_logs"("user_id");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_user_id" ON "ai_logs"("user_id");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_search_vector" ON "ai_logs" USING GIN("search_vector");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_model" ON "ai_logs"("model");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_provider" ON "ai_logs"("provider_id");

--bun:split
CREATE OR REPLACE FUNCTION prevent_ai_logs_modification()
    RETURNS TRIGGER
    AS $$
BEGIN
    RAISE EXCEPTION 'Modifications are not allowed on ai_logs (append-only table)';
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER enforce_ai_logs_append_only
    BEFORE UPDATE OR DELETE ON "ai_logs"
    FOR EACH ROW
    EXECUTE FUNCTION prevent_ai_logs_modification();

--bun:split
COMMENT ON TABLE "ai_logs" IS 'Stores logs for AI operations';

COMMENT ON COLUMN "ai_logs"."model" IS 'Free-text model identifier as reported by the provider; not a closed set because self-hosted models are named arbitrarily';

COMMENT ON COLUMN "ai_logs"."provider_kind" IS 'Wire protocol that served the call (AnthropicMessages, OpenAIResponses, OpenAIChat)';
