-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" DROP CONSTRAINT "accounting_controls_organizations_accounting_control", DROP COLUMN "organization_accounting_control", ADD COLUMN "organization_id" uuid NOT NULL, ADD CONSTRAINT "accounting_controls_organizations_accounting_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "accounting_controls_organization_id_key" to table: "accounting_controls"
CREATE UNIQUE INDEX "accounting_controls_organization_id_key" ON "accounting_controls" ("organization_id");
