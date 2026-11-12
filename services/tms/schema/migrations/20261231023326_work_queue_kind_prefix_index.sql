--keel:notx
-- The backstop sweep lists due work by a kind prefix (one kind per agent):
-- a LIKE prefix needs pattern ops to use the index under a non-C collation.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_work_queue_items_kind_prefix_due" ON "work_queue_items"("kind" varchar_pattern_ops, "due_at");
