-- Permission System V2 - Materialized Views Rollback
-- Remove materialized views and related functions

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_policies_refresh ON policies;
DROP TRIGGER IF EXISTS trigger_roles_refresh ON roles;
DROP TRIGGER IF EXISTS trigger_user_memberships_refresh ON user_organization_memberships;
DROP TRIGGER IF EXISTS trigger_user_roles_refresh ON user_organization_roles;

-- Drop functions
DROP FUNCTION IF EXISTS trigger_refresh_user_policies();
DROP FUNCTION IF EXISTS refresh_user_permissions(VARCHAR(100), VARCHAR(100));
DROP FUNCTION IF EXISTS refresh_user_effective_policies();

-- Drop indexes (will be dropped with materialized view, but explicit for clarity)
DROP INDEX IF EXISTS idx_user_effective_policies_unique;
DROP INDEX IF EXISTS idx_user_effective_policies_user_org;
DROP INDEX IF EXISTS idx_user_effective_policies_effect_priority;
DROP INDEX IF EXISTS idx_user_effective_policies_assignment_type;
DROP INDEX IF EXISTS idx_user_effective_policies_hash;

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS user_effective_policies;