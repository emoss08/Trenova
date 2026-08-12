-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260526143000_centralized_iam.tx.up.sql

CREATE TABLE IF NOT EXISTS identity_providers(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "slug" TEXT NOT NULL,
    "protocol" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "enforce_sso" INTEGER NOT NULL DEFAULT 0,
    "auto_provision" INTEGER NOT NULL DEFAULT 0,
    "allow_federated_mfa" INTEGER NOT NULL DEFAULT 1,
    "allowed_domains" TEXT NOT NULL,
    "attribute_map" TEXT NOT NULL DEFAULT '{}',
    "oidc_issuer_url" TEXT,
    "oidc_client_id" TEXT,
    "oidc_client_secret" TEXT,
    "oidc_redirect_url" TEXT,
    "oidc_scopes" TEXT NOT NULL,
    "saml_entity_id" TEXT,
    "saml_sso_url" TEXT,
    "saml_x509_certificate" TEXT,
    "saml_metadata_xml" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (organization_id, slug),
    CONSTRAINT identity_provider_oidc_required CHECK (protocol != 'OIDC' OR (oidc_issuer_url IS NOT NULL AND oidc_client_id IS NOT NULL AND oidc_redirect_url IS NOT NULL)),
    CONSTRAINT identity_provider_saml_required CHECK (protocol != 'SAML' OR (saml_entity_id IS NOT NULL AND saml_sso_url IS NOT NULL AND saml_x509_certificate IS NOT NULL))
);

--bun:split

CREATE TABLE IF NOT EXISTS external_identities(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "identity_provider_id" TEXT NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
    "external_subject" TEXT NOT NULL,
    "external_username" TEXT,
    "external_email" TEXT,
    "raw_claims" TEXT NOT NULL DEFAULT '{}',
    "last_login_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (identity_provider_id, external_subject),
    UNIQUE (organization_id, user_id, identity_provider_id)
);

--bun:split

CREATE TABLE IF NOT EXISTS mfa_authenticators(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "type" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "credential_id" TEXT,
    "secret_cipher" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "verified_at" INTEGER,
    "last_used_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT mfa_authenticator_material CHECK (credential_id IS NOT NULL OR secret_cipher IS NOT NULL)
);

--bun:split

CREATE TABLE IF NOT EXISTS risk_decisions(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT,
    "organization_id" TEXT,
    "business_unit_id" TEXT,
    "outcome" TEXT NOT NULL,
    "signals" TEXT NOT NULL,
    "reason" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE TABLE IF NOT EXISTS auth_events(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT,
    "organization_id" TEXT,
    "business_unit_id" TEXT,
    "identity_provider_id" TEXT,
    "provider" TEXT NOT NULL,
    "outcome" TEXT NOT NULL,
    "ip_address" TEXT,
    "user_agent" TEXT,
    "authenticator_aal" INTEGER NOT NULL DEFAULT 1,
    "federation_fal" INTEGER NOT NULL DEFAULT 1,
    "mfa_state" TEXT,
    "risk_outcome" TEXT NOT NULL DEFAULT 'allow',
    "risk_signals" TEXT NOT NULL,
    "error_code" TEXT,
    "occurred_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE TABLE IF NOT EXISTS scim_directories(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "tenant_slug" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (tenant_slug)
);

--bun:split

CREATE TABLE IF NOT EXISTS scim_tokens(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "directory_id" TEXT NOT NULL REFERENCES scim_directories(id) ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "prefix" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'active',
    "last_used_at" INTEGER,
    "expires_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (prefix),
    UNIQUE (token_hash)
);

--bun:split

CREATE TABLE IF NOT EXISTS "scim_group_role_mappings"(
    "id" TEXT,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "directory_id" TEXT NOT NULL,
    "external_group_id" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "role_id" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_scim_group_role_mappings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_sgrm_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sgrm_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sgrm_directory" FOREIGN KEY ("directory_id") REFERENCES "scim_directories"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_sgrm_role" FOREIGN KEY ("role_id", "business_unit_id", "organization_id") REFERENCES "roles"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_sgrm_display_name" ON "scim_group_role_mappings" (lower("display_name"), "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS provisioning_audit_records(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "directory_id" TEXT NOT NULL REFERENCES scim_directories(id) ON DELETE CASCADE,
    "action" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "external_id" TEXT,
    "resource_id" TEXT,
    "status" TEXT NOT NULL,
    "error_message" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE TABLE IF NOT EXISTS access_policies(
    "id" TEXT PRIMARY KEY,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "resource" TEXT NOT NULL,
    "operation" TEXT NOT NULL,
    "effect" TEXT NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "conditions" TEXT NOT NULL DEFAULT '{}',
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_identity_providers_org_enabled ON identity_providers (organization_id, enabled);

--bun:split

CREATE INDEX IF NOT EXISTS idx_external_identities_user_org ON external_identities (user_id, organization_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_mfa_authenticators_user_org_enabled ON mfa_authenticators (user_id, organization_id, enabled);

--bun:split

CREATE INDEX IF NOT EXISTS idx_auth_events_org_user_time ON auth_events (organization_id, user_id, occurred_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_auth_events_ip_time ON auth_events (ip_address, occurred_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_risk_decisions_org_user_time ON risk_decisions (organization_id, user_id, created_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_scim_tokens_directory_status ON scim_tokens (directory_id, status);

--bun:split

CREATE INDEX IF NOT EXISTS idx_scim_group_role_mappings_directory_group ON scim_group_role_mappings (directory_id, external_group_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_provisioning_audit_directory_time ON provisioning_audit_records (directory_id, created_at DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_access_policies_org_resource ON access_policies (organization_id, resource, operation, enabled, priority DESC);
