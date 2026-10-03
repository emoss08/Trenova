-- When an item stops being something a person can still head off: a
-- credential expiring, a move starting, a proposal expiring. The tower orders
-- itself by it; an item with no deadline has none.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "due_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "due_label" VARCHAR(120);

--bun:split
-- A person putting an item aside until later. Only for them: everyone else
-- still sees it, and the item itself is untouched.
CREATE TABLE IF NOT EXISTS "watchtower_snoozes" (
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "item_id" VARCHAR(100) NOT NULL,
    "until" BIGINT NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    PRIMARY KEY ("organization_id", "business_unit_id", "user_id", "item_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_snoozes_user_until"
    ON "watchtower_snoozes" ("organization_id", "business_unit_id", "user_id", "until");
