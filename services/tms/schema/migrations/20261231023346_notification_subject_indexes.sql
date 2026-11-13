--keel:notx
-- One open row per subject per person: the conflict a second notice about the same subject lands on.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_notifications_open_subject" ON "notifications"("organization_id", "target_user_id", "subject_key") WHERE "dismissed_at" IS NULL AND "subject_key" <> '' AND "target_user_id" IS NOT NULL;

--keel:split
-- A person's open notifications in the order the panel shows them, and the counts on its tabs.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_notifications_open" ON "notifications"("organization_id", "target_user_id", "surfaces_at" DESC, "id" DESC) WHERE "dismissed_at" IS NULL;

--keel:split
-- What a person finished and what cleared itself, newest first.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_notifications_done" ON "notifications"("organization_id", "target_user_id", "surfaces_at" DESC, "id" DESC) WHERE "dismissed_at" IS NOT NULL;

--keel:split
-- Every person's open row for a subject, to close them together when the work is finished.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_notifications_subject" ON "notifications"("organization_id", "subject_key") WHERE "dismissed_at" IS NULL AND "subject_key" <> '';

--keel:split
-- The panel no longer reads by read state alone, so nothing uses these.
DROP INDEX CONCURRENTLY IF EXISTS "idx_notifications_user";

--keel:split
DROP INDEX CONCURRENTLY IF EXISTS "idx_notifications_unread";
