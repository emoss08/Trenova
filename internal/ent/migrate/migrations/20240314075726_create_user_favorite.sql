-- Modify "users" table
ALTER TABLE "users" ALTER COLUMN "timezone" SET DEFAULT 'AmericaLosAngeles';
-- Create "user_favorites" table
CREATE TABLE "user_favorites" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "page_link" character varying NOT NULL, "user_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "user_favorites_users_user_favorites" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- Create index "user_favorites_page_link_key" to table: "user_favorites"
CREATE UNIQUE INDEX "user_favorites_page_link_key" ON "user_favorites" ("page_link");
