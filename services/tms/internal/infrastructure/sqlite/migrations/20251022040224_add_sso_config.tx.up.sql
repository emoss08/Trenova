-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251022040224_add_sso_config.tx.up.sql

CREATE TABLE IF NOT EXISTS "sso_configs"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "protocol" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "enforce_sso" INTEGER NOT NULL DEFAULT 0,
    "auto_provision" INTEGER NOT NULL DEFAULT 1,
    "default_role" TEXT,
    "allowed_domains" TEXT,
    "attribute_map" TEXT,
    "oidc_issuer_url" TEXT NOT NULL,
    "oidc_client_id" TEXT NOT NULL,
    "oidc_client_secret" TEXT NOT NULL,
    "oidc_redirect_url" TEXT NOT NULL,
    "oidc_scopes" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_sso_configs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_sso_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sso_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_sso_configs_name_org" UNIQUE ("organization_id", "name")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sso_configs_business_unit" ON "sso_configs" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sso_configs_created_at" ON "sso_configs" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sso_configs_enabled" ON "sso_configs" ("organization_id", "enabled")WHERE
    "enabled" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_sso_configs_provider" ON "sso_configs" ("provider", "organization_id");
