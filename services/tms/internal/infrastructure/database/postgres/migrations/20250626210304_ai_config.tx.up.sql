--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "ai_configs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "api_key" varchar(255) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_ai_configs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ai_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one ai config per organization
    CONSTRAINT "uq_ai_configs_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_configs_business_unit" ON "ai_configs"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_ai_configs_created_at" ON "ai_configs"("created_at", "updated_at");

-- Add comment to describe the table purpose
COMMENT ON TABLE ai_configs IS 'Stores configuration for AI configs and validation rules';

--bun:split
CREATE OR REPLACE FUNCTION ai_configs_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS ai_configs_update_timestamp_trigger ON ai_configs;

CREATE TRIGGER ai_configs_update_timestamp_trigger
    BEFORE UPDATE ON ai_configs
    FOR EACH ROW
    EXECUTE FUNCTION ai_configs_update_timestamp();

--bun:split
ALTER TABLE ai_configs
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE ai_configs
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

