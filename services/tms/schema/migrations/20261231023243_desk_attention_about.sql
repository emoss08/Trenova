-- The Desk's Needs you groups what the Watchtower holds by the business record each item is
-- about: a shipment's late warning, the customer's email about it and the plan to recover it
-- are one row. An item's subject is the record an agent would work on, which for a message or a
-- decision is the message or the decision itself; the record it is about is stored beside it,
-- once, by the source that knows it, so grouping never works it out per read.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "about_kind" varchar(64),
    ADD COLUMN IF NOT EXISTS "about_id" varchar(100);

ALTER TABLE "watchtower_items"
    ADD CONSTRAINT "ck_watchtower_items_about_pair" CHECK (("about_kind" IS NULL) = ("about_id" IS NULL));

COMMENT ON COLUMN "watchtower_items"."about_kind" IS 'Record kind of the business record the item is about; set together with about_id';
COMMENT ON COLUMN "watchtower_items"."about_id" IS 'Id of the business record the item is about, which the Desk groups by';

-- A plan is about the record most of its steps change, kept when the plan opens so the
-- Watchtower can group it without reading its steps.
ALTER TABLE "agent_plans"
    ADD COLUMN IF NOT EXISTS "target_id" varchar(100);

COMMENT ON COLUMN "agent_plans"."target_id" IS 'The record most of the plan''s steps change; null when no step names one';
