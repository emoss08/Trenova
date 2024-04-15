-- Drop index "general_ledger_accounts_account_number_key" from table: "general_ledger_accounts"
DROP INDEX "general_ledger_accounts_account_number_key";
-- Create index "generalledgeraccount_account_number_organization_id" to table: "general_ledger_accounts"
CREATE UNIQUE INDEX "generalledgeraccount_account_number_organization_id" ON "general_ledger_accounts" ("account_number", "organization_id");
-- Create "tags" table
CREATE TABLE "tags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying NOT NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "tags_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tags_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "tag_name_organization_id" to table: "tags"
CREATE UNIQUE INDEX "tag_name_organization_id" ON "tags" ("name", "organization_id");
