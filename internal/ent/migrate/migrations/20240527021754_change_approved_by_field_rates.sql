-- Rename a column from "approved_by" to "approved_by_id"
ALTER TABLE "rates" RENAME COLUMN "approved_by" TO "approved_by_id";

-- Modify "rates" table
ALTER TABLE "rates" ADD CONSTRAINT "rates_users_rates_approved" FOREIGN KEY ("approved_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
