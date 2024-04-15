-- Modify "user_favorites" table
ALTER TABLE "user_favorites" DROP CONSTRAINT "user_favorites_users_user_favorites", ALTER COLUMN "user_id" SET NOT NULL, ADD CONSTRAINT "user_favorites_users_user_favorites" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
