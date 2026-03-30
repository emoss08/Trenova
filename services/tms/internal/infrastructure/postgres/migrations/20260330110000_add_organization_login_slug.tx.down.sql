DROP INDEX IF EXISTS "idx_organizations_login_slug";

--bun:split
ALTER TABLE "organizations"
    DROP COLUMN IF EXISTS "login_slug";
