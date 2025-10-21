--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TYPE "email_provider_type_enum" AS ENUM(
    'SMTP',
    'Resend'
);

--bun:split
CREATE TYPE "email_auth_type_enum" AS ENUM(
    'Plain',
    'Login',
    'CRAMMD5',
    'OAuth2',
    'APIKey'
);

--bun:split
CREATE TYPE "email_encryption_type_enum" AS ENUM(
    'None',
    'SSL_TLS',
    'StartTLS'
);

--bun:split
CREATE TYPE "email_template_category_enum" AS ENUM(
    'Transactional',
    'Notification',
    'Marketing',
    'System',
    'Custom'
);

--bun:split
CREATE TYPE "email_priority_enum" AS ENUM(
    'High',
    'Medium',
    'Low'
);

--bun:split
CREATE TYPE "email_queue_status_enum" AS ENUM(
    'Pending',
    'Processing',
    'Sent',
    'Failed',
    'Scheduled',
    'Cancelled'
);

--bun:split
CREATE TYPE "email_log_status_enum" AS ENUM(
    'Delivered',
    'Opened',
    'Clicked',
    'Bounced',
    'Complained',
    'Unsubscribed',
    'Rejected'
);

--bun:split
CREATE TYPE "email_bounce_type_enum" AS ENUM(
    'Hard',
    'Soft',
    'Block'
);

--bun:split
CREATE TABLE IF NOT EXISTS "email_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "is_default" boolean DEFAULT FALSE,
    "status" status_enum DEFAULT 'Active',
    "provider_type" email_provider_type_enum NOT NULL,
    "auth_type" email_auth_type_enum NOT NULL,
    "encryption_type" email_encryption_type_enum NOT NULL,
    "host" varchar(255),
    "port" integer,
    "username" varchar(255),
    "encrypted_password" text,
    "encrypted_api_key" text,
    "oauth2_client_id" varchar(255),
    "oauth2_client_secret" text,
    "oauth2_tenant_id" varchar(255),
    "from_address" varchar(255) NOT NULL,
    "from_name" varchar(255),
    "reply_to" varchar(255),
    "max_connections" integer DEFAULT 5,
    "timeout_seconds" integer DEFAULT 30,
    "retry_count" integer DEFAULT 3,
    "retry_delay_seconds" integer DEFAULT 5,
    "rate_limit_per_minute" integer DEFAULT 60,
    "rate_limit_per_hour" integer DEFAULT 1000,
    "rate_limit_per_day" integer DEFAULT 10000,
    "metadata" jsonb DEFAULT '{}',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    -- Provider-specific validations
    CONSTRAINT "chk_email_profiles_smtp_config" CHECK (CASE WHEN "provider_type" = 'SMTP' THEN
        "host" IS NOT NULL AND "port" IS NOT NULL AND ("auth_type" != 'APIKey' OR "encrypted_api_key" IS NOT NULL)
    ELSE
        TRUE
    END),
    CONSTRAINT "chk_email_profiles_api_providers" CHECK (CASE WHEN "provider_type" IN ('Resend') THEN
        "encrypted_api_key" IS NOT NULL
    ELSE
        TRUE
    END),
    CONSTRAINT "chk_email_profiles_oauth_providers" CHECK (CASE WHEN "provider_type" IN ('Resend') AND "auth_type" = 'OAuth2' THEN
        "oauth2_client_id" IS NOT NULL AND "oauth2_client_secret" IS NOT NULL AND "oauth2_tenant_id" IS NOT NULL
    ELSE
        TRUE
    END)
);

--bun:split
-- Add table comment for email_profiles
COMMENT ON TABLE "email_profiles" IS 'Stores email configuration profiles for different providers, allowing organizations to manage multiple email sending configurations';

--bun:split
-- Add column comments for email_profiles
COMMENT ON COLUMN "email_profiles"."id" IS 'Unique identifier for the email profile';

COMMENT ON COLUMN "email_profiles"."business_unit_id" IS 'Reference to the business unit this profile belongs to';

COMMENT ON COLUMN "email_profiles"."organization_id" IS 'Reference to the organization this profile belongs to';

COMMENT ON COLUMN "email_profiles"."name" IS 'Human-readable name for the email profile';

COMMENT ON COLUMN "email_profiles"."description" IS 'Optional description of the email profile purpose or usage';

COMMENT ON COLUMN "email_profiles"."is_default" IS 'Flag indicating if this is the default profile for the organization';

COMMENT ON COLUMN "email_profiles"."status" IS 'Active/Inactive status of the profile';

COMMENT ON COLUMN "email_profiles"."provider_type" IS 'Type of email provider (SMTP, Resend, etc.)';

COMMENT ON COLUMN "email_profiles"."auth_type" IS 'Authentication method used by the provider';

COMMENT ON COLUMN "email_profiles"."encryption_type" IS 'Encryption method for secure connections';

COMMENT ON COLUMN "email_profiles"."host" IS 'SMTP server hostname (for SMTP providers)';

COMMENT ON COLUMN "email_profiles"."port" IS 'SMTP server port number';

COMMENT ON COLUMN "email_profiles"."username" IS 'Username for authentication';

COMMENT ON COLUMN "email_profiles"."encrypted_password" IS 'Encrypted password for authentication';

COMMENT ON COLUMN "email_profiles"."encrypted_api_key" IS 'Encrypted API key for API-based providers';

COMMENT ON COLUMN "email_profiles"."oauth2_client_id" IS 'OAuth2 client ID for OAuth2 authentication';

COMMENT ON COLUMN "email_profiles"."oauth2_client_secret" IS 'OAuth2 client secret (encrypted)';

COMMENT ON COLUMN "email_profiles"."oauth2_tenant_id" IS 'OAuth2 tenant ID for Microsoft services';

COMMENT ON COLUMN "email_profiles"."from_address" IS 'Default sender email address';

COMMENT ON COLUMN "email_profiles"."from_name" IS 'Default sender display name';

COMMENT ON COLUMN "email_profiles"."reply_to" IS 'Default reply-to email address';

COMMENT ON COLUMN "email_profiles"."max_connections" IS 'Maximum concurrent connections allowed';

COMMENT ON COLUMN "email_profiles"."timeout_seconds" IS 'Connection timeout in seconds';

COMMENT ON COLUMN "email_profiles"."retry_count" IS 'Number of retry attempts for failed sends';

COMMENT ON COLUMN "email_profiles"."retry_delay_seconds" IS 'Delay between retry attempts in seconds';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_minute" IS 'Maximum emails allowed per minute';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_hour" IS 'Maximum emails allowed per hour';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_day" IS 'Maximum emails allowed per day';

COMMENT ON COLUMN "email_profiles"."metadata" IS 'Additional provider-specific configuration stored as JSON';

COMMENT ON COLUMN "email_profiles"."version" IS 'Version number for optimistic locking';

COMMENT ON COLUMN "email_profiles"."created_at" IS 'Unix timestamp when the record was created';

COMMENT ON COLUMN "email_profiles"."updated_at" IS 'Unix timestamp when the record was last updated';

--bun:split
-- Create indexes for email profiles
CREATE INDEX "idx_email_profiles_org_id" ON "email_profiles"("organization_id");

CREATE INDEX "idx_email_profiles_business_unit_id" ON "email_profiles"("business_unit_id");

CREATE INDEX "idx_email_profiles_is_default" ON "email_profiles"("is_default")
WHERE
    "is_default" = TRUE;

CREATE INDEX "idx_email_profile_bu_org_id" ON "email_profiles"("business_unit_id", "organization_id", "id");

CREATE INDEX "idx_email_profiles_status" ON "email_profiles"("status")
WHERE
    "status" = 'Active';

CREATE INDEX "idx_email_profiles_provider_type" ON "email_profiles"("provider_type");

CREATE UNIQUE INDEX "uniq_email_profiles_org_name" ON "email_profiles"("organization_id", LOWER("name"));

CREATE INDEX "idx_email_profiles_created_at" ON "email_profiles"("created_at" DESC);

-- Composite index for common queries
CREATE INDEX "idx_email_profiles_org_status_default" ON "email_profiles"("organization_id", "status", "is_default")
WHERE
    "status" = 'Active';

--bun:split
-- Set statistics for frequently queried columns
ALTER TABLE "email_profiles"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "email_profiles"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

ALTER TABLE "email_profiles"
    ALTER COLUMN "status" SET STATISTICS 500;

-- Search Vector SQL for EmailProfile
--bun:split
ALTER TABLE "email_profiles"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_email_profiles_search_vector ON "email_profiles" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION email_profiles_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.from_address, '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.host, '')), 'C') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'D');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS email_profiles_search_update ON "email_profiles";

CREATE TRIGGER email_profiles_search_update
    BEFORE INSERT OR UPDATE ON "email_profiles"
    FOR EACH ROW
    EXECUTE FUNCTION email_profiles_search_trigger();

--bun:split
-- Function to ensure only one default email profile per organization
CREATE OR REPLACE FUNCTION "ensure_single_default_email_profile"()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.is_default = TRUE THEN
        UPDATE
            email_profiles
        SET
            is_default = FALSE,
            updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
        WHERE
            organization_id = NEW.organization_id
            AND id != NEW.id
            AND is_default = TRUE;
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER "email_profiles_default_check"
    AFTER INSERT OR UPDATE OF is_default ON email_profiles
    FOR EACH ROW
    WHEN(NEW.is_default = TRUE)
    EXECUTE FUNCTION ensure_single_default_email_profile();

