DROP TABLE IF EXISTS "locations" CASCADE;

-- bun:split

DROP TABLE IF EXISTS "location_comments" CASCADE;

-- bun:split

DROP TABLE IF EXISTS "location_contacts" CASCADE;

-- bun:split

DROP INDEX IF EXISTS "locations_code_organization_id_unq" CASCADE;
DROP INDEX IF EXISTS "idx_location_name" CASCADE;
DROP INDEX IF EXISTS "idx_location_org_bu" CASCADE;
DROP INDEX IF EXISTS "idx_location_description" CASCADE;
DROP INDEX IF EXISTS "idx_location_created_at" CASCADE;
DROP INDEX IF EXISTS "idx_location_comment_comment_type_id" CASCADE;
DROP INDEX IF EXISTS "idx_location_comment_created_at" CASCADE;
DROP INDEX IF EXISTS "idx_location_comment_created_at" CASCADE;
DROP INDEX IF EXISTS "idx_location_contact_name" CASCADE;
DROP INDEX IF EXISTS "idx_location_contact_created_at" CASCADE;
