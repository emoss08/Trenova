-- Modify "organizations" table
ALTER TABLE "organizations" DROP COLUMN "organization_accounting_control";
-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" DROP CONSTRAINT "accounting_controls_business_units_business_unit", DROP COLUMN "organization_id", ADD COLUMN "business_unit_id" uuid NOT NULL, ADD COLUMN "organization_accounting_control" uuid NOT NULL, ADD CONSTRAINT "accounting_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "accounting_controls_organizations_accounting_control" FOREIGN KEY ("organization_accounting_control") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "accounting_controls_organization_accounting_control_key" to table: "accounting_controls"
CREATE UNIQUE INDEX "accounting_controls_organization_accounting_control_key" ON "accounting_controls" ("organization_accounting_control");
