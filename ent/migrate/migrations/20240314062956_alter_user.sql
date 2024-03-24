-- Modify "users" table
ALTER TABLE "users"
ADD COLUMN "organization_users" uuid NULL,
ADD CONSTRAINT "users_organizations_users" FOREIGN KEY ("organization_users") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;