-- Modify "organizations" table
ALTER TABLE "organizations" DROP CONSTRAINT "organizations_business_units_organizations", ALTER COLUMN "business_unit_organizations" SET NOT NULL, ADD CONSTRAINT "organizations_business_units_organizations" FOREIGN KEY ("business_unit_organizations") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
