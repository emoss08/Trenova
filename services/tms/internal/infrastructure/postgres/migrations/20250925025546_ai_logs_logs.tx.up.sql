CREATE TYPE "operation_enum" AS ENUM(
    'ClassifyLocation'
);

CREATE TYPE "model_enum" AS ENUM(
    'gpt-5-nano',
    'gpt-5-nano-2025-08-07',
    'omni-moderation-latest'
);

CREATE TABLE IF NOT EXISTS "ai_logs"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "prompt" text NOT NULL,
    "response" text NOT NULL,
    "model" model_enum NOT NULL,
    "operation" operation_enum NOT NULL,
    "object" varchar(100) NOT NULL,
    "service_tier" varchar(100) NOT NULL,
    "prompt_tokens" integer NOT NULL,
    "completion_tokens" integer NOT NULL,
    "total_tokens" integer NOT NULL,
    "reasoning_tokens" integer NOT NULL,
    "timestamp" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_ai_logs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_logs_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_logs_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_logs_timestamp" ON "ai_logs"("timestamp");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_org_bu" ON "ai_logs"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_user" ON "ai_logs"("user_id");

--bun:split
-- Function to prevent updates/deletes on ai_logs
CREATE OR REPLACE FUNCTION prevent_ai_logs_modification()
    RETURNS TRIGGER
    AS $$
BEGIN
    RAISE EXCEPTION 'Modifications are not allowed on ai_logs (append-only table)';
END;
$$
LANGUAGE plpgsql;

-- Trigger to enforce append-only behavior
CREATE TRIGGER enforce_ai_logs_append_only
    BEFORE UPDATE OR DELETE ON "ai_logs"
    FOR EACH ROW
    EXECUTE FUNCTION prevent_ai_logs_modification();

COMMENT ON TABLE "ai_logs" IS 'Stores logs for AI operations';

