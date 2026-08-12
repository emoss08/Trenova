-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250927053802_permission_system_v2.tx.up.sql

CREATE TABLE IF NOT EXISTS policies(
    "id" TEXT PRIMARY KEY,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "scope" TEXT NOT NULL,
    "resources" TEXT,
    "subjects" TEXT,
    "effect" TEXT NOT NULL,
    "priority" INTEGER,
    "tags" TEXT,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE TABLE IF NOT EXISTS roles(
    "id" TEXT PRIMARY KEY,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "level" TEXT NOT NULL,
    "parent_roles" TEXT,
    "scope" TEXT NOT NULL,
    "policy_ids" TEXT,
    "auto_assign" TEXT,
    "is_system" INTEGER,
    "is_admin" INTEGER,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

DROP TABLE IF EXISTS user_organizations;

--bun:split

CREATE TABLE IF NOT EXISTS user_organization_memberships(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "role_ids" TEXT,
    "direct_policies" TEXT,
    "joined_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "granted_by_id" TEXT,
    "expires_at" INTEGER,
    "is_default" INTEGER,
    UNIQUE (user_id, organization_id)
);

--bun:split

CREATE TABLE IF NOT EXISTS user_organization_roles(
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "role_id" TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    "assigned_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "assigned_by" TEXT,
    "expires_at" INTEGER,
    "overrides" TEXT,
    PRIMARY KEY (user_id, organization_id, role_id),
    FOREIGN KEY (user_id, organization_id) REFERENCES user_organization_memberships(user_id, organization_id) ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS permission_cache(
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "computed_at" INTEGER NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "permission_data" BLOB NOT NULL,
    "bloom_filter" BLOB,
    "checksum" TEXT NOT NULL,
    PRIMARY KEY (user_id, organization_id)
);

--bun:split

CREATE TABLE IF NOT EXISTS policy_templates(
    "id" TEXT PRIMARY KEY,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "industry" TEXT,
    "category" TEXT,
    "policies" TEXT NOT NULL DEFAULT '[]',
    "role_structure" TEXT NOT NULL DEFAULT '[]',
    "is_active" INTEGER,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_policies_business_unit ON policies ((scope -> 'businessUnitId'));

--bun:split

CREATE INDEX IF NOT EXISTS idx_policies_effect_priority ON policies (effect, priority DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_roles_business_unit ON roles (business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_memberships_user_id ON user_organization_memberships (user_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_memberships_org_id ON user_organization_memberships (organization_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_memberships_is_default ON user_organization_memberships (is_default)WHERE
    is_default = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_roles_user_org ON user_organization_roles (user_id, organization_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_roles_role_id ON user_organization_roles (role_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_org_roles_expires_at ON user_organization_roles (expires_at)WHERE
    expires_at IS NOT NULL;
