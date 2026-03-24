-- RBAC V3 Rollback
-- Drops V3 tables and restores V2 structure

-- Drop V3 tables
DROP TABLE IF EXISTS user_role_assignments CASCADE;
DROP TABLE IF EXISTS resource_permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;

-- Remove is_platform_admin from users
ALTER TABLE users DROP COLUMN IF EXISTS is_platform_admin;

-- Restore columns to user_organization_memberships
ALTER TABLE user_organization_memberships
    ADD COLUMN IF NOT EXISTS role_ids TEXT[] DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS direct_policies TEXT[] DEFAULT ARRAY[]::TEXT[];

-- Note: V2 tables (policies, roles, etc.) would need to be recreated
-- by running the V2 migration again if needed
