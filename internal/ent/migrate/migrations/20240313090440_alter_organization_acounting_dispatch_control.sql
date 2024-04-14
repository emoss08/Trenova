-- Modify "organizations" table
ALTER TABLE "organizations" DROP COLUMN "organization_billing_control", DROP COLUMN "organization_dispatch_control";
-- Modify "billing_controls" table
ALTER TABLE "billing_controls" DROP CONSTRAINT "billing_controls_organizations_organization", ADD CONSTRAINT "billing_controls_organizations_billing_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "billing_controls_organization_id_key" to table: "billing_controls"
CREATE UNIQUE INDEX "billing_controls_organization_id_key" ON "billing_controls" ("organization_id");
-- Modify "dispatch_controls" table
ALTER TABLE "dispatch_controls" DROP CONSTRAINT "dispatch_controls_organizations_organization", ADD CONSTRAINT "dispatch_controls_organizations_dispatch_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "dispatch_controls_organization_id_key" to table: "dispatch_controls"
CREATE UNIQUE INDEX "dispatch_controls_organization_id_key" ON "dispatch_controls" ("organization_id");
