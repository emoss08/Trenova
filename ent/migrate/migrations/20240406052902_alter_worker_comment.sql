-- Modify "worker_comments" table
ALTER TABLE "worker_comments" DROP COLUMN "entered_by", ADD COLUMN "user_id" uuid NOT NULL, ADD CONSTRAINT "worker_comments_users_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
