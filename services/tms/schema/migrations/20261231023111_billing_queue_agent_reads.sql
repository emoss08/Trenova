-- What Nova read to review an item: the sections it was given and the reads it made, so the
-- biller can see what a note rests on.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "agent_reads" jsonb;
