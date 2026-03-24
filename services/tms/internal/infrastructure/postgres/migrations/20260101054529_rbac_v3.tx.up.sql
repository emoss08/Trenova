-- RBAC V3 Migration
-- Replaces the complex policy-based V2 system with a simpler role-based system
-- Drop V2 triggers first
DROP TRIGGER IF EXISTS trigger_policies_refresh ON policies;

DROP TRIGGER IF EXISTS trigger_roles_refresh ON roles;

DROP TRIGGER IF EXISTS trigger_user_memberships_refresh ON user_organization_memberships;

DROP TRIGGER IF EXISTS trigger_user_roles_refresh ON user_organization_roles;

-- Drop V2 functions
DROP FUNCTION IF EXISTS trigger_refresh_user_policies();

DROP FUNCTION IF EXISTS refresh_user_permissions(varchar, varchar);

DROP FUNCTION IF EXISTS refresh_user_effective_policies();

-- Drop V2 materialized view
DROP MATERIALIZED VIEW IF EXISTS user_effective_policies;

-- Drop V2 tables
DROP TABLE IF EXISTS policy_templates CASCADE;

DROP TABLE IF EXISTS permission_cache CASCADE;

DROP TABLE IF EXISTS user_organization_roles CASCADE;

DROP TABLE IF EXISTS roles CASCADE;

DROP TABLE IF EXISTS policies CASCADE;

-- Drop V2 enum types
DROP TYPE IF EXISTS policy_condition_type_enum CASCADE;

DROP TYPE IF EXISTS subject_type_enum CASCADE;

DROP TYPE IF EXISTS mask_type_enum CASCADE;

DROP TYPE IF EXISTS scope_type_enum CASCADE;

DROP TYPE IF EXISTS role_level_enum CASCADE;

DROP TYPE IF EXISTS data_scope_enum CASCADE;

DROP TYPE IF EXISTS effect_enum CASCADE;

--bun:split
-- Remove V2 columns from user_organization_memberships
ALTER TABLE user_organization_memberships
    DROP COLUMN IF EXISTS role_ids,
    DROP COLUMN IF EXISTS direct_policies;

--bun:split
-- Add is_platform_admin to users table
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_platform_admin boolean NOT NULL DEFAULT FALSE;

--bun:split
-- Create roles table
CREATE TABLE roles(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "parent_role_ids" text[] DEFAULT ARRAY[] ::text[],
    "max_sensitivity" varchar(20) NOT NULL DEFAULT 'internal',
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "is_org_admin" boolean NOT NULL DEFAULT FALSE,
    "created_by" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_roles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_roles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_roles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_roles_created_by" FOREIGN KEY ("created_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX "uniq_roles_id" ON roles("id");

--bun:split
CREATE UNIQUE INDEX "uniq_roles_name_org" ON roles(lower("name"), "organization_id");

--bun:split
CREATE TABLE resource_permissions(
    "id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "resource" varchar(100) NOT NULL,
    "operations" text[] NOT NULL DEFAULT ARRAY[] ::text[],
    "data_scope" varchar(20) NOT NULL DEFAULT 'organization',
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_resource_permissions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_resource_permissions_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE TABLE IF NOT EXISTS user_role_assignments(
    "id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "expires_at" bigint,
    "assigned_by" varchar(100),
    "assigned_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_user_role_assignments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_user_role_assignments_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_assigned_by" FOREIGN KEY ("assigned_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_org ON user_role_assignments("user_id", "organization_id");

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role_id ON user_role_assignments("role_id");

