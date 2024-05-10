-- Modify "permissions" table
ALTER TABLE "permissions" DROP CONSTRAINT "permissions_resources_permissions", DROP COLUMN "resource_permissions", ADD COLUMN "resource_id" uuid NOT NULL, ADD CONSTRAINT "permissions_resources_permissions" FOREIGN KEY ("resource_id") REFERENCES "resources" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
