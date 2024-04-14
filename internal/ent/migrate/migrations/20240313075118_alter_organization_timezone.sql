-- Modify "organizations" table
ALTER TABLE "organizations" DROP CONSTRAINT "organizations_accounting_controls_accounting_control", ALTER COLUMN "organization_accounting_control" DROP NOT NULL, ADD CONSTRAINT "organizations_accounting_controls_accounting_control" FOREIGN KEY ("organization_accounting_control") REFERENCES "accounting_controls" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
