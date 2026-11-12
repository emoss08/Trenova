-- What is happening on an item, in the shape made for it: a stop and its
-- window, a detention clock, a paper that is missing, a message, credentials
-- coming due. The source fills it as it describes the item; an item with
-- none is read from its facts.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "panel" jsonb;

COMMENT ON COLUMN "watchtower_items"."panel" IS 'What is happening, typed by kind (Stop, Clock, Paper, Message, Expiry, Coverage, Contact, Run), for the view made for that kind; null when the item is read from its facts';
