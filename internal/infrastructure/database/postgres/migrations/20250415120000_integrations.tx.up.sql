CREATE TYPE integration_type AS ENUM(
    'GoogleMaps',
    'PCMiler'
);

--bun:split
CREATE TYPE integration_category AS ENUM(
    'MappingRouting',
    'FreightLogistics'
);

--bun:split
CREATE TABLE IF NOT EXISTS "integrations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Integration details
    "type" integration_type NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "built_by" varchar(100),
    -- UI and documentation enhancements
    "overview" text,
    "screenshots" jsonb DEFAULT '[]' ::jsonb,
    "features" jsonb DEFAULT '[]' ::jsonb,
    "category" integration_category NOT NULL,
    -- Type-specific configuration fields
    "config_fields" jsonb DEFAULT '{}' ::jsonb,
    "event_triggers" jsonb DEFAULT '[]' ::jsonb,
    "webhook_endpoints" jsonb DEFAULT '[]' ::jsonb,
    -- Configuration stored as JSON
    "configuration" jsonb DEFAULT '{}' ::jsonb,
    -- Usage statistics
    "last_used" bigint,
    "usage_count" bigint NOT NULL DEFAULT 0,
    "error_count" bigint NOT NULL DEFAULT 0,
    "last_error" text,
    "last_error_at" bigint,
    "enabled_by_id" varchar(100),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_integrations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_integrations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_integrations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_integrations_enabled_by" FOREIGN KEY ("enabled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_integrations_organization_type" UNIQUE ("organization_id", "business_unit_id", "type")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_integrations_business_unit" ON "integrations"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_integrations_type" ON "integrations"("type");

CREATE INDEX IF NOT EXISTS "idx_integrations_created_at" ON "integrations"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_integrations_configuration" ON "integrations" USING gin("configuration");

CREATE INDEX IF NOT EXISTS "idx_integrations_config_fields" ON "integrations" USING gin("config_fields");

CREATE INDEX IF NOT EXISTS "idx_integrations_event_triggers" ON "integrations" USING gin("event_triggers");

-- Add comment to describe the table purpose
COMMENT ON TABLE integrations IS 'Stores configuration for external service integrations';

--bun:split
CREATE OR REPLACE FUNCTION integrations_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS "integrations_update_timestamp_trigger" ON "integrations";

CREATE TRIGGER "integrations_update_timestamp_trigger"
    BEFORE UPDATE ON "integrations"
    FOR EACH ROW
    EXECUTE FUNCTION "integrations_update_timestamp"();

--bun:split
ALTER TABLE "integrations"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "integrations"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

