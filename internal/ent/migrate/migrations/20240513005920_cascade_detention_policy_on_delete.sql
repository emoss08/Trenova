-- Modify "customer_detention_policies" table
ALTER TABLE "customer_detention_policies" DROP CONSTRAINT "customer_detention_policies_customers_detention_policies", ADD CONSTRAINT "customer_detention_policies_customers_detention_policies" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
