SET statement_timeout = 0;

-- Create SSO Protocol enum type
CREATE TYPE sso_protocol_enum AS ENUM(
    'OIDC'
);

--bun:split
-- Create SSO Provider enum type
CREATE TYPE sso_provider_enum AS ENUM(
    'Okta',
    'AzureAD',
    'Auth0',
    'Google',
    'GenericOIDC'
);

--bun:split
-- Create SSO Config table
CREATE TABLE IF NOT EXISTS "sso_configs"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "provider" sso_provider_enum NOT NULL,
    "protocol" sso_protocol_enum NOT NULL,
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "enforce_sso" boolean NOT NULL DEFAULT FALSE,
    "auto_provision" boolean NOT NULL DEFAULT TRUE,
    "default_role" varchar(50),
    "allowed_domains" text[],
    "attribute_map" jsonb,
    -- OIDC Configuration
    "oidc_issuer_url" varchar(500) NOT NULL,
    "oidc_client_id" varchar(255) NOT NULL,
    "oidc_client_secret" text NOT NULL,
    "oidc_redirect_url" varchar(500) NOT NULL,
    "oidc_scopes" text[] NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_sso_configs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sso_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sso_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure unique SSO config name per organization
    CONSTRAINT "uq_sso_configs_name_org" UNIQUE ("organization_id", "name")
);

--bun:split
-- General business unit and organization index
CREATE INDEX IF NOT EXISTS "idx_sso_configs_business_unit" ON "sso_configs"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sso_configs_created_at" ON "sso_configs"("created_at", "updated_at");

--bun:split
-- Index for enabled SSO configs (for fast lookup during authentication)
CREATE INDEX IF NOT EXISTS "idx_sso_configs_enabled" ON "sso_configs"("organization_id", "enabled")
WHERE
    "enabled" = TRUE;

--bun:split
-- Index for SSO provider lookups
CREATE INDEX IF NOT EXISTS "idx_sso_configs_provider" ON "sso_configs"("provider", "organization_id");

--bun:split
COMMENT ON TABLE sso_configs IS 'Stores Single Sign-On (SSO) configurations for organizations using OIDC protocol';

--bun:split
COMMENT ON COLUMN sso_configs.oidc_client_secret IS 'Encrypted OAuth2 client secret - must be encrypted before storage using AES-256-GCM';

--bun:split
COMMENT ON COLUMN sso_configs.attribute_map IS 'Maps SSO provider attributes to user fields (e.g., {"email": "email", "firstName": "given_name"})';

--bun:split
COMMENT ON COLUMN sso_configs.allowed_domains IS 'Email domains allowed for SSO authentication (e.g., ["company.com", "contractor.com"]). Empty array allows all domains.';

--bun:split
COMMENT ON COLUMN sso_configs.enforce_sso IS 'When true, disables password-based authentication for users in this organization';

--bun:split
-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION sso_configs_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS sso_configs_update_trigger ON sso_configs;

--bun:split
CREATE TRIGGER sso_configs_update_trigger
    BEFORE UPDATE ON sso_configs
    FOR EACH ROW
    EXECUTE FUNCTION sso_configs_update_timestamps();

--bun:split
-- Set statistics for query planner optimization
ALTER TABLE sso_configs
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE sso_configs
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE sso_configs
    ALTER COLUMN provider SET STATISTICS 1000;

--bun:split
-- Add full-text search vector column
ALTER TABLE "sso_configs"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_sso_configs_search_vector ON "sso_configs" USING GIN(search_vector);

--bun:split
-- Create trigger function for updating search vector
CREATE OR REPLACE FUNCTION sso_configs_update_search_vector()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.provider::text, '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.oidc_issuer_url, '')), 'C');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS sso_configs_search_vector_trigger ON sso_configs;

--bun:split
CREATE TRIGGER sso_configs_search_vector_trigger
    BEFORE INSERT OR UPDATE ON sso_configs
    FOR EACH ROW
    EXECUTE FUNCTION sso_configs_update_search_vector();

--bun:split
-- Update existing rows to populate search_vector (if any exist)
UPDATE
    sso_configs
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(provider::text, '')), 'B') || setweight(to_tsvector('english', COALESCE(oidc_issuer_url, '')), 'C')
WHERE
    search_vector IS NULL;

