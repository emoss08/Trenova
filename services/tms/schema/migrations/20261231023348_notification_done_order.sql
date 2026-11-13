--keel:notx
-- Done lists what a person finished and what cleared itself by when it was finished, so the
-- index that serves it is ordered by that time rather than by when the row last surfaced.
DROP INDEX CONCURRENTLY IF EXISTS "idx_notifications_done";

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_notifications_done" ON "notifications"("organization_id", "target_user_id", "dismissed_at" DESC, "id" DESC) WHERE "dismissed_at" IS NOT NULL;
