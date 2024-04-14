-- Modify "shipments" table
ALTER TABLE "shipments" DROP CONSTRAINT "shipments_users_created_by_user", ADD CONSTRAINT "shipments_users_shipments" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
