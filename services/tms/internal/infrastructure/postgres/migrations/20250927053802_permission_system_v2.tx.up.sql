CREATE TYPE effect_enum AS ENUM(
    'allow',
    'deny'
);

CREATE TYPE data_scope_enum AS ENUM(
    'own',
    'organization',
    'business_unit',
    'all'
);

CREATE TYPE role_level_enum AS ENUM(
    'system',
    'bu',
    'org',
    'custom'
);

CREATE TYPE scope_type_enum AS ENUM(
    'business_unit',
    'organization'
);

CREATE TYPE mask_type_enum AS ENUM(
    'partial',
    'full',
    'hash'
);

CREATE TYPE subject_type_enum AS ENUM(
    'user',
    'role'
);

CREATE TYPE policy_condition_type_enum AS ENUM(
    'field',
    'time',
    'ip',
    'attribute'
);

--bun:split
CREATE TABLE policies(
    id varchar(100) PRIMARY KEY,
    name varchar(255) NOT NULL,
    description text,
    scope jsonb NOT NULL,
    resources jsonb DEFAULT '[]' ::jsonb,
    subjects jsonb DEFAULT '[]' ::jsonb,
    effect effect_enum NOT NULL,
    priority int DEFAULT 0,
    tags text[] DEFAULT ARRAY[] ::text[],
    created_by varchar(100),
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint
);

--bun:split
CREATE TABLE roles(
    id varchar(100) PRIMARY KEY,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    name varchar(255) NOT NULL,
    description text,
    level role_level_enum NOT NULL,
    parent_roles text[] DEFAULT ARRAY[] ::text[],
    scope jsonb NOT NULL,
    policy_ids text[] DEFAULT ARRAY[] ::text[],
    auto_assign jsonb,
    is_system boolean DEFAULT FALSE,
    is_admin boolean DEFAULT FALSE,
    created_by varchar(100),
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint
);

--bun:split
DROP TABLE IF EXISTS user_organizations CASCADE;

CREATE TABLE user_organization_memberships(
    id varchar(100) PRIMARY KEY,
    user_id varchar(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    organization_id varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role_ids text[] DEFAULT ARRAY[] ::text[],
    direct_policies text[] DEFAULT ARRAY[] ::text[],
    joined_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    granted_by_id varchar(100) REFERENCES users(id) ON DELETE CASCADE,
    expires_at bigint,
    is_default boolean DEFAULT FALSE,
    UNIQUE (user_id, organization_id)
);

--bun:split
CREATE TABLE user_organization_roles(
    user_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    role_id varchar(100) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    assigned_by varchar(100),
    expires_at bigint,
    overrides jsonb DEFAULT '[]' ::jsonb,
    PRIMARY KEY (user_id, organization_id, role_id),
    FOREIGN KEY (user_id, organization_id) REFERENCES user_organization_memberships(user_id, organization_id) ON DELETE CASCADE
);

--bun:split
CREATE TABLE permission_cache(
    user_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    version varchar(50) NOT NULL,
    computed_at bigint NOT NULL,
    expires_at bigint NOT NULL,
    permission_data bytea NOT NULL,
    bloom_filter bytea,
    checksum varchar(64) NOT NULL,
    PRIMARY KEY (user_id, organization_id)
);

--bun:split
CREATE TABLE policy_templates(
    id varchar(100) PRIMARY KEY,
    name varchar(255) NOT NULL,
    description text,
    industry varchar(100),
    category varchar(100),
    policies jsonb NOT NULL DEFAULT '[]' ::jsonb,
    role_structure jsonb NOT NULL DEFAULT '[]' ::jsonb,
    is_active boolean DEFAULT TRUE,
    created_by varchar(100),
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint
);

--bun:split
CREATE INDEX idx_policies_business_unit ON policies((scope -> 'businessUnitId'));

CREATE INDEX idx_policies_effect_priority ON policies(effect, priority DESC);

CREATE INDEX idx_roles_business_unit ON roles(business_unit_id);

CREATE INDEX idx_roles_policy_ids ON roles USING GIN(policy_ids);

CREATE INDEX idx_user_org_memberships_user_id ON user_organization_memberships(user_id);

CREATE INDEX idx_user_org_memberships_org_id ON user_organization_memberships(organization_id);

CREATE INDEX idx_user_org_memberships_is_default ON user_organization_memberships(is_default)
WHERE
    is_default = TRUE;

CREATE INDEX idx_user_org_roles_user_org ON user_organization_roles(user_id, organization_id);

CREATE INDEX idx_user_org_roles_role_id ON user_organization_roles(role_id);

CREATE INDEX idx_user_org_roles_expires_at ON user_organization_roles(expires_at)
WHERE
    expires_at IS NOT NULL;

