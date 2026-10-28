--keel:notx
-- A chapter starts at the first message its opening turn saved, so reading a conversation's
-- chapters finds that message by thread and turn without walking the thread.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_messages_turn" ON "assistant_messages" ("organization_id", "business_unit_id", "thread_id", "turn_id", "sequence") WHERE "turn_id" IS NOT NULL;
