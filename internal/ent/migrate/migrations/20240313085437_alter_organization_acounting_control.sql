-- Create index "organization_business_unit_id_scac_code" to table: "organizations"
CREATE UNIQUE INDEX "organization_business_unit_id_scac_code" ON "organizations" ("business_unit_id", "scac_code");
-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" DROP CONSTRAINT "accounting_controls_organizations_accounting_control", DROP COLUMN "organization_id", ADD COLUMN "organization_accounting_control" uuid NULL, ADD CONSTRAINT "accounting_controls_organizations_accounting_control" FOREIGN KEY ("organization_accounting_control") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "accounting_controls_organization_accounting_control_key" to table: "accounting_controls"
CREATE UNIQUE INDEX "accounting_controls_organization_accounting_control_key" ON "accounting_controls" ("organization_accounting_control");
