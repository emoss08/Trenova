DROP TABLE IF EXISTS "customers" CASCADE;

-- bun:split

DROP INDEX IF EXISTS "customers_code_organization_id_unq" CASCADE;
DROP INDEX IF EXISTS "idx_customer_name" CASCADE;
DROP INDEX IF EXISTS "idx_customer_org_bu" CASCADE;
DROP INDEX IF EXISTS "idx_customer_created_at" CASCADE;
