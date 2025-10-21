-- Permission System V2 Rollback
-- Remove all new permission system tables and types
-- Drop triggers
DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;

DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes (will be dropped with tables, but explicit for clarity)
DROP INDEX IF EXISTS idx_policies_business_unit;

DROP INDEX IF EXISTS idx_policies_effect_priority;

DROP INDEX IF EXISTS idx_policies_created_at;

DROP INDEX IF EXISTS idx_policies_tags;

DROP INDEX IF EXISTS idx_roles_business_unit;

DROP INDEX IF EXISTS idx_roles_level;

DROP INDEX IF EXISTS idx_roles_is_system;

DROP INDEX IF EXISTS idx_roles_policy_ids;

DROP INDEX IF EXISTS idx_user_org_memberships_user_id;

DROP INDEX IF EXISTS idx_user_org_memberships_org_id;

DROP INDEX IF EXISTS idx_user_org_memberships_is_default;

DROP INDEX IF EXISTS idx_user_org_memberships_expires_at;

DROP INDEX IF EXISTS idx_user_org_roles_user_org;

DROP INDEX IF EXISTS idx_user_org_roles_role_id;

DROP INDEX IF EXISTS idx_user_org_roles_expires_at;

DROP INDEX IF EXISTS idx_permission_cache_expires_at;

DROP INDEX IF EXISTS idx_permission_cache_computed_at;

DROP INDEX IF EXISTS idx_permission_cache_version;

DROP INDEX IF EXISTS idx_policy_templates_industry;

DROP INDEX IF EXISTS idx_policy_templates_category;

DROP INDEX IF EXISTS idx_policy_templates_is_active;

DROP TABLE IF EXISTS policy_templates;

DROP TABLE IF EXISTS permission_cache;

DROP TABLE IF EXISTS user_organization_roles;

DROP TABLE IF EXISTS user_organization_memberships;

DROP TABLE IF EXISTS roles;

DROP TABLE IF EXISTS policies;

-- Drop enum types
DROP TYPE IF EXISTS policy_condition_type_enum;

DROP TYPE IF EXISTS subject_type_enum;

DROP TYPE IF EXISTS mask_type_enum;

DROP TYPE IF EXISTS scope_type_enum;

DROP TYPE IF EXISTS role_level_enum;

DROP TYPE IF EXISTS data_scope_enum;

DROP TYPE IF EXISTS effect_enum;

-- Recreate old user_organizations table if needed
CREATE TABLE IF NOT EXISTS user_organizations(
    user_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    created_at bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    PRIMARY KEY (user_id, organization_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

