-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250712224201_add_email_profile.tx.up.sql

CREATE TABLE IF NOT EXISTS "email_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_default" INTEGER,
    "status" TEXT,
    "provider_type" TEXT NOT NULL,
    "auth_type" TEXT NOT NULL,
    "encryption_type" TEXT NOT NULL,
    "host" TEXT,
    "port" INTEGER,
    "username" TEXT,
    "encrypted_password" TEXT,
    "encrypted_api_key" TEXT,
    "oauth2_client_id" TEXT,
    "oauth2_client_secret" TEXT,
    "oauth2_tenant_id" TEXT,
    "from_address" TEXT NOT NULL,
    "from_name" TEXT,
    "reply_to" TEXT,
    "max_connections" INTEGER,
    "timeout_seconds" INTEGER,
    "retry_count" INTEGER,
    "retry_delay_seconds" INTEGER,
    "rate_limit_per_minute" INTEGER,
    "rate_limit_per_hour" INTEGER,
    "rate_limit_per_day" INTEGER,
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS "idx_email_profiles_org_id" ON "email_profiles" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_business_unit_id" ON "email_profiles" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_is_default" ON "email_profiles" ("is_default")WHERE
    "is_default" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profile_bu_org_id" ON "email_profiles" ("business_unit_id", "organization_id", "id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_status" ON "email_profiles" ("status")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_provider_type" ON "email_profiles" ("provider_type");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_email_profiles_org_name" ON "email_profiles" ("organization_id", LOWER("name"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_created_at" ON "email_profiles" ("created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_email_profiles_org_status_default" ON "email_profiles" ("organization_id", "status", "is_default")WHERE
    "status" = 'Active';
