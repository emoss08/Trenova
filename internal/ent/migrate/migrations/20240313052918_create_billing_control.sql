-- Modify "organizations" table
ALTER TABLE "organizations" ADD COLUMN "organization_billing_control" uuid NULL, ADD CONSTRAINT "organizations_billing_controls_billing_control" FOREIGN KEY ("organization_billing_control") REFERENCES "billing_controls" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
