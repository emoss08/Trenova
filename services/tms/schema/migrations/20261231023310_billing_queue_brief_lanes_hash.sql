ALTER TABLE "billing_queue_briefs"
    ADD COLUMN IF NOT EXISTS "lanes_hash" varchar(64) NOT NULL DEFAULT '';
