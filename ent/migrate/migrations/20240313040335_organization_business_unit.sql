-- Modify "organizations" table
ALTER TABLE "organizations" DROP CONSTRAINT "organizations_business_units_business_unit", ADD COLUMN "business_unit_organizations" uuid NULL, ADD CONSTRAINT "organizations_business_units_organizations" FOREIGN KEY ("business_unit_organizations") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
