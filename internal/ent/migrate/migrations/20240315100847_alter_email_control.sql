-- Create "email_controls" table
CREATE TABLE "email_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "email_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "email_controls_organizations_email_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "email_controls_organization_id_key" to table: "email_controls"
CREATE UNIQUE INDEX "email_controls_organization_id_key" ON "email_controls" ("organization_id");
