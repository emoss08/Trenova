-- Modify "shipments" table
ALTER TABLE "shipments" DROP CONSTRAINT "shipments_customers_customer", ADD CONSTRAINT "shipments_customers_shipments" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
