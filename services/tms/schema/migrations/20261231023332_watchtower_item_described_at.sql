-- When what an item says last changed: its title, summary, facts, impact,
-- trail or panel. An agent handed the item reads that description, so one
-- that looked and proposed nothing is not started on it again until the
-- description is newer than its hand-off. Items raised before this column
-- count as described when they were raised.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "described_at" bigint NOT NULL DEFAULT 0;

UPDATE "watchtower_items"
SET "described_at" = "created_at"
WHERE "described_at" = 0;

COMMENT ON COLUMN "watchtower_items"."described_at" IS 'Unix seconds when the item''s description (title, summary, facts, impact, trail, panel) last changed';
