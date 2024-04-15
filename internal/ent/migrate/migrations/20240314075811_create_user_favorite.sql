-- Modify "user_favorites" table
ALTER TABLE "user_favorites"
ADD COLUMN "business_unit_id" uuid NOT NULL,
ADD COLUMN "organization_id" uuid NOT NULL,
ADD CONSTRAINT "user_favorites_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
ADD CONSTRAINT "user_favorites_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;