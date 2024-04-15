-- Modify "business_units" table
ALTER TABLE "business_units" DROP CONSTRAINT "business_units_business_units_parent", ADD CONSTRAINT "business_units_business_units_next" FOREIGN KEY ("parent_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
