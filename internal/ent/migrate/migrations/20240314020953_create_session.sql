-- Drop index "organization_business_unit_id_scac_code" from table: "organizations"
DROP INDEX "organization_business_unit_id_scac_code";
-- Modify "table_change_alerts" table
ALTER TABLE "table_change_alerts" DROP COLUMN "conditional_logic";
-- Modify "users" table
ALTER TABLE "users" DROP COLUMN "date_joined", ADD COLUMN "last_login" timestamptz NULL;
-- Create index "user_username_email" to table: "users"
CREATE UNIQUE INDEX "user_username_email" ON "users" ("username", "email");
-- Create "sessions" table
CREATE TABLE "sessions" ("id" character varying NOT NULL, "data" character varying NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "expires_at" timestamptz NOT NULL, PRIMARY KEY ("id"));
