-- Modify "organizations" table
ALTER TABLE "organizations" DROP CONSTRAINT "organizations_business_units_organizations", DROP COLUMN "business_unit_organizations", ADD CONSTRAINT "organizations_business_units_organizations" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
