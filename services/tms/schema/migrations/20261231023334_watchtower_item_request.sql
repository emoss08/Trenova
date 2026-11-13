-- The last play a person sent for an item (asking for a proof of delivery,
-- asking where a load is, offering a move to carriers again): who asked,
-- when, and what they asked in a sentence. The case shows it so the next
-- person sees the question is already out.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "last_request" jsonb;

COMMENT ON COLUMN "watchtower_items"."last_request" IS 'The last play a person sent for the item: {at, byId, byName, summary}; null when none has been';

ALTER TABLE "watchtower_item_events"
    DROP CONSTRAINT IF EXISTS "ck_watchtower_item_events_kind";

ALTER TABLE "watchtower_item_events"
    ADD CONSTRAINT "ck_watchtower_item_events_kind" CHECK ("kind" IN (
        'Created', 'Assigned', 'Unassigned', 'Acknowledged', 'Started', 'Snoozed', 'Woke',
        'DueChanged', 'Commented', 'Escalated', 'Resolved', 'Reopened', 'SourceResolved',
        'SourceReopened', 'HandedOff', 'Requested'
    ));
