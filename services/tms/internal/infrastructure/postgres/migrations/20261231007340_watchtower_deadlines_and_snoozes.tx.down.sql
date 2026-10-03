DROP TABLE IF EXISTS "watchtower_snoozes";

--bun:split
ALTER TABLE "watchtower_items"
    DROP COLUMN IF EXISTS "due_label",
    DROP COLUMN IF EXISTS "due_at";
