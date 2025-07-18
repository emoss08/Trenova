-- Drop triggers first
DROP TRIGGER IF EXISTS notifications_update_trigger ON notifications;

--bun:split
-- Drop trigger function
DROP FUNCTION IF EXISTS notifications_update_timestamps();

--bun:split
-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_organization;

--bun:split
DROP INDEX IF EXISTS idx_notifications_user;

--bun:split
DROP INDEX IF EXISTS idx_notifications_role;

--bun:split
DROP INDEX IF EXISTS idx_notifications_unread;

--bun:split
DROP INDEX IF EXISTS idx_notifications_delivery;

--bun:split
DROP INDEX IF EXISTS idx_notifications_cleanup;

--bun:split
DROP INDEX IF EXISTS idx_notifications_job;

--bun:split
DROP INDEX IF EXISTS idx_notifications_event_type;

--bun:split
-- Drop the table (this will cascade and remove all foreign key references)
DROP TABLE IF EXISTS notifications;
