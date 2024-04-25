-- Modify "permissions" table
ALTER TABLE "permissions" DROP COLUMN "resource", ADD COLUMN "resource_permissions" uuid NULL, ADD CONSTRAINT "permissions_resources_permissions" FOREIGN KEY ("resource_permissions") REFERENCES "resources" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
