-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" DROP CONSTRAINT "accounting_controls_business_units_business_unit", DROP COLUMN "business_unit_id", ADD CONSTRAINT "accounting_controls_business_units_business_unit" FOREIGN KEY ("organization_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
