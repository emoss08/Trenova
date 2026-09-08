DROP INDEX IF EXISTS "idx_user_organization_memberships_position";

--bun:split
ALTER TABLE IF EXISTS "user_organization_memberships"
    DROP CONSTRAINT IF EXISTS "fk_user_organization_memberships_position";

--bun:split
ALTER TABLE IF EXISTS "user_organization_memberships"
    DROP COLUMN IF EXISTS "position_id";
