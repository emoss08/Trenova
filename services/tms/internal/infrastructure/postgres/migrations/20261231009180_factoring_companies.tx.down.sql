ALTER TABLE "invoices" DROP CONSTRAINT IF EXISTS "fk_invoices_factoring_company";
DROP INDEX IF EXISTS "idx_invoices_factoring_company";
ALTER TABLE "invoices" DROP COLUMN IF EXISTS "factoring_company_id";
ALTER TABLE "customer_billing_profiles" DROP CONSTRAINT IF EXISTS "fk_customer_billing_profiles_factoring_company";
DROP INDEX IF EXISTS "idx_customer_billing_profiles_factoring_company";
ALTER TABLE "customer_billing_profiles" DROP COLUMN IF EXISTS "factoring_company_id";
DROP TABLE IF EXISTS "factoring_companies";
