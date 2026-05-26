CREATE TYPE iam_identity_provider_protocol_enum AS ENUM('OIDC', 'SAML');
CREATE TYPE iam_mfa_authenticator_type_enum AS ENUM('webauthn', 'totp');
CREATE TYPE iam_auth_event_outcome_enum AS ENUM('success', 'challenge', 'denied', 'failed');
CREATE TYPE iam_risk_outcome_enum AS ENUM('allow', 'challenge', 'deny');
CREATE TYPE iam_scim_token_status_enum AS ENUM('active', 'revoked');
CREATE TYPE iam_provisioning_action_enum AS ENUM('create', 'update', 'deactivate', 'delete');
CREATE TYPE iam_policy_effect_enum AS ENUM('allow', 'deny');

--bun:split
CREATE TABLE identity_providers(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    name varchar(120) NOT NULL,
    slug varchar(80) NOT NULL,
    protocol iam_identity_provider_protocol_enum NOT NULL,
    enabled boolean NOT NULL DEFAULT TRUE,
    enforce_sso boolean NOT NULL DEFAULT FALSE,
    auto_provision boolean NOT NULL DEFAULT FALSE,
    allow_federated_mfa boolean NOT NULL DEFAULT TRUE,
    allowed_domains text[] NOT NULL DEFAULT ARRAY[]::text[],
    attribute_map jsonb NOT NULL DEFAULT '{}'::jsonb,
    oidc_issuer_url varchar(500),
    oidc_client_id varchar(255),
    oidc_client_secret text,
    oidc_redirect_url varchar(500),
    oidc_scopes text[] NOT NULL DEFAULT ARRAY['openid', 'email', 'profile']::text[],
    saml_entity_id varchar(500),
    saml_sso_url varchar(500),
    saml_x509_certificate text,
    saml_metadata_xml text,
    version bigint NOT NULL DEFAULT 0,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    UNIQUE (organization_id, slug),
    CONSTRAINT identity_provider_oidc_required CHECK (
        protocol != 'OIDC'
        OR (oidc_issuer_url IS NOT NULL AND oidc_client_id IS NOT NULL AND oidc_redirect_url IS NOT NULL)
    ),
    CONSTRAINT identity_provider_saml_required CHECK (
        protocol != 'SAML'
        OR (saml_entity_id IS NOT NULL AND saml_sso_url IS NOT NULL AND saml_x509_certificate IS NOT NULL)
    )
);

--bun:split
INSERT INTO identity_providers(
    id,
    organization_id,
    business_unit_id,
    name,
    slug,
    protocol,
    enabled,
    enforce_sso,
    auto_provision,
    allowed_domains,
    attribute_map,
    oidc_issuer_url,
    oidc_client_id,
    oidc_client_secret,
    oidc_redirect_url,
    oidc_scopes,
    version,
    created_at,
    updated_at
)
SELECT
    id,
    organization_id,
    business_unit_id,
    name,
    lower(provider::text),
    'OIDC',
    enabled,
    enforce_sso,
    auto_provision,
    COALESCE(allowed_domains, ARRAY[]::text[]),
    COALESCE(attribute_map, '{}'::jsonb),
    oidc_issuer_url,
    oidc_client_id,
    oidc_client_secret,
    oidc_redirect_url,
    COALESCE(oidc_scopes, ARRAY['openid', 'email', 'profile']::text[]),
    version,
    created_at,
    updated_at
FROM sso_configs
ON CONFLICT (organization_id, slug) DO NOTHING;

--bun:split
CREATE TABLE external_identities(
    id varchar(100) PRIMARY KEY,
    user_id varchar(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    identity_provider_id varchar(100) NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
    external_subject varchar(255) NOT NULL,
    external_username varchar(255),
    external_email varchar(255),
    raw_claims jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_login_at bigint,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    UNIQUE (identity_provider_id, external_subject),
    UNIQUE (organization_id, user_id, identity_provider_id)
);

--bun:split
CREATE TABLE mfa_authenticators(
    id varchar(100) PRIMARY KEY,
    user_id varchar(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type iam_mfa_authenticator_type_enum NOT NULL,
    name varchar(120) NOT NULL,
    credential_id varchar(512),
    secret_cipher text,
    enabled boolean NOT NULL DEFAULT FALSE,
    verified_at bigint,
    last_used_at bigint,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    CONSTRAINT mfa_authenticator_material CHECK (
        credential_id IS NOT NULL
        OR secret_cipher IS NOT NULL
    )
);

--bun:split
CREATE TABLE risk_decisions(
    id varchar(100) PRIMARY KEY,
    user_id varchar(100) REFERENCES users(id) ON DELETE SET NULL,
    organization_id varchar(100) REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) REFERENCES business_units(id) ON DELETE CASCADE,
    outcome iam_risk_outcome_enum NOT NULL,
    signals text[] NOT NULL DEFAULT ARRAY[]::text[],
    reason text,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint
);

--bun:split
CREATE TABLE auth_events(
    id varchar(100) PRIMARY KEY,
    user_id varchar(100) REFERENCES users(id) ON DELETE SET NULL,
    organization_id varchar(100) REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) REFERENCES business_units(id) ON DELETE CASCADE,
    identity_provider_id varchar(100) REFERENCES identity_providers(id) ON DELETE SET NULL,
    provider varchar(120) NOT NULL,
    outcome iam_auth_event_outcome_enum NOT NULL,
    ip_address inet,
    user_agent text,
    authenticator_aal int NOT NULL DEFAULT 1,
    federation_fal int NOT NULL DEFAULT 1,
    mfa_state varchar(80),
    risk_outcome iam_risk_outcome_enum NOT NULL DEFAULT 'allow',
    risk_signals text[] NOT NULL DEFAULT ARRAY[]::text[],
    error_code varchar(120),
    occurred_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint
);

--bun:split
CREATE TABLE scim_directories(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    tenant_slug varchar(100) NOT NULL,
    enabled boolean NOT NULL DEFAULT TRUE,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    UNIQUE (tenant_slug)
);

--bun:split
CREATE TABLE scim_tokens(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    directory_id varchar(100) NOT NULL REFERENCES scim_directories(id) ON DELETE CASCADE,
    name varchar(120) NOT NULL,
    prefix varchar(24) NOT NULL,
    token_hash varchar(128) NOT NULL,
    status iam_scim_token_status_enum NOT NULL DEFAULT 'active',
    last_used_at bigint,
    expires_at bigint,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    UNIQUE (prefix),
    UNIQUE (token_hash)
);

--bun:split
CREATE TABLE scim_group_role_mappings(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    directory_id varchar(100) NOT NULL REFERENCES scim_directories(id) ON DELETE CASCADE,
    external_group_id varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    role_id varchar(100) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    UNIQUE (directory_id, external_group_id, role_id)
);

--bun:split
CREATE TABLE provisioning_audit_records(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    directory_id varchar(100) NOT NULL REFERENCES scim_directories(id) ON DELETE CASCADE,
    action iam_provisioning_action_enum NOT NULL,
    resource_type varchar(80) NOT NULL,
    external_id varchar(255),
    resource_id varchar(100),
    status varchar(80) NOT NULL,
    error_message text,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint
);

--bun:split
CREATE TABLE access_policies(
    id varchar(100) PRIMARY KEY,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    name varchar(120) NOT NULL,
    resource varchar(120) NOT NULL,
    operation varchar(80) NOT NULL,
    effect iam_policy_effect_enum NOT NULL,
    priority int NOT NULL DEFAULT 0,
    conditions jsonb NOT NULL DEFAULT '{}'::jsonb,
    enabled boolean NOT NULL DEFAULT TRUE,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp)::bigint
);

--bun:split
CREATE INDEX idx_identity_providers_org_enabled ON identity_providers(organization_id, enabled);
CREATE INDEX idx_external_identities_user_org ON external_identities(user_id, organization_id);
CREATE INDEX idx_mfa_authenticators_user_org_enabled ON mfa_authenticators(user_id, organization_id, enabled);
CREATE INDEX idx_auth_events_org_user_time ON auth_events(organization_id, user_id, occurred_at DESC);
CREATE INDEX idx_auth_events_ip_time ON auth_events(ip_address, occurred_at DESC);
CREATE INDEX idx_risk_decisions_org_user_time ON risk_decisions(organization_id, user_id, created_at DESC);
CREATE INDEX idx_scim_tokens_directory_status ON scim_tokens(directory_id, status);
CREATE INDEX idx_scim_group_role_mappings_directory_group ON scim_group_role_mappings(directory_id, external_group_id);
CREATE INDEX idx_provisioning_audit_directory_time ON provisioning_audit_records(directory_id, created_at DESC);
CREATE INDEX idx_access_policies_org_resource ON access_policies(organization_id, resource, operation, enabled, priority DESC);
