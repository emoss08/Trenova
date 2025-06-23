CREATE TABLE IF NOT EXISTS "notifications"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Optional targeting
    "business_unit_id" varchar(100),
    "target_user_id" varchar(100),
    "target_role_id" varchar(100),
    -- Notification metadata
    "event_type" varchar(100) NOT NULL,
    "priority" varchar(20) NOT NULL DEFAULT 'medium',
    "channel" varchar(20) NOT NULL DEFAULT 'global',
    -- Content
    "title" varchar(255) NOT NULL,
    "message" text NOT NULL,
    "data" jsonb,
    "related_entities" jsonb,
    "actions" jsonb,
    -- Delivery & Lifecycle
    "expires_at" bigint,
    "delivered_at" bigint,
    "read_at" bigint,
    "dismissed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Retry & Tracking
    "delivery_status" varchar(20) NOT NULL DEFAULT 'pending',
    "retry_count" int NOT NULL DEFAULT 0,
    "max_retries" int NOT NULL DEFAULT 3,
    -- Metadata
    "source" varchar(100) NOT NULL,
    "job_id" varchar(255),
    "correlation_id" varchar(255),
    "tags" text[],
    -- Version for optimistic locking
    "version" bigint NOT NULL DEFAULT 0,
    -- Constraints
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_notifications_organization_id" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notifications_business_unit_id" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notifications_target_user_id" FOREIGN KEY ("target_user_id") REFERENCES "users"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notifications_target_role_id" FOREIGN KEY ("target_role_id", "business_unit_id", "organization_id") REFERENCES "roles"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    -- Check constraints for enums
    CONSTRAINT "chk_notifications_priority" CHECK ("priority" IN ('critical', 'high', 'medium', 'low')),
    CONSTRAINT "chk_notifications_channel" CHECK ("channel" IN ('global', 'user', 'role')),
    CONSTRAINT "chk_notifications_delivery_status" CHECK ("delivery_status" IN ('pending', 'delivered', 'failed', 'expired')),
    -- Business logic constraints
    CONSTRAINT "chk_notifications_retry_count" CHECK ("retry_count" >= 0 AND "retry_count" <= "max_retries"),
    CONSTRAINT "chk_notifications_max_retries" CHECK ("max_retries" >= 0 AND "max_retries" <= 10),
    -- Channel-specific targeting constraints
    CONSTRAINT "chk_notifications_user_channel" CHECK (("channel" = 'user' AND "target_user_id" IS NOT NULL) OR ("channel" != 'user')),
    CONSTRAINT "chk_notifications_role_channel" CHECK (("channel" = 'role' AND "target_role_id" IS NOT NULL AND "business_unit_id" IS NOT NULL) OR ("channel" != 'role'))
);

--bun:split
-- Index for organization-based queries (most common)
CREATE INDEX IF NOT EXISTS "idx_notifications_organization" ON "notifications"("organization_id", "created_at" DESC);

--bun:split
-- Index for user-specific notifications
CREATE INDEX IF NOT EXISTS "idx_notifications_user" ON "notifications"("target_user_id", "organization_id", "read_at", "created_at" DESC)
WHERE
    "target_user_id" IS NOT NULL;

--bun:split
-- Index for role-based notifications
CREATE INDEX IF NOT EXISTS "idx_notifications_role" ON "notifications"("target_role_id", "business_unit_id", "organization_id", "created_at" DESC)
WHERE
    "target_role_id" IS NOT NULL;

--bun:split
-- Index for unread notifications (common query)
CREATE INDEX IF NOT EXISTS "idx_notifications_unread" ON "notifications"("organization_id", "read_at", "expires_at", "created_at" DESC)
WHERE
    "read_at" IS NULL;

--bun:split
-- Index for delivery status and retry logic
CREATE INDEX IF NOT EXISTS "idx_notifications_delivery" ON "notifications"("delivery_status", "retry_count", "max_retries", "expires_at")
WHERE
    "delivery_status" IN ('failed', 'pending');

--bun:split
-- Index for cleanup queries (expired/old notifications)
CREATE INDEX IF NOT EXISTS "idx_notifications_cleanup" ON "notifications"("created_at", "read_at", "dismissed_at", "expires_at");

--bun:split
-- Index for job tracking
CREATE INDEX IF NOT EXISTS "idx_notifications_job" ON "notifications"("job_id", "source", "created_at" DESC)
WHERE
    "job_id" IS NOT NULL;

--bun:split
-- Index for event type analytics
CREATE INDEX IF NOT EXISTS "idx_notifications_event_type" ON "notifications"("event_type", "organization_id", "created_at" DESC);

--bun:split
-- Comments for documentation
COMMENT ON TABLE notifications IS 'Stores real-time notifications for users with multi-tenant isolation and WebSocket delivery tracking';

COMMENT ON COLUMN notifications.channel IS 'Notification channel: global (all org users), user (specific user), role (users with role)';

COMMENT ON COLUMN notifications.priority IS 'Notification priority: critical, high, medium, low';

COMMENT ON COLUMN notifications.delivery_status IS 'WebSocket delivery status: pending, delivered, failed, expired';

COMMENT ON COLUMN notifications.event_type IS 'Hierarchical event type (e.g., job.shipment.duplicate_complete)';

COMMENT ON COLUMN notifications.data IS 'Additional structured data for the notification';

COMMENT ON COLUMN notifications.related_entities IS 'Array of related entities (shipments, workers, etc.)';

COMMENT ON COLUMN notifications.actions IS 'Array of action buttons/links for interactive notifications';

COMMENT ON COLUMN notifications.tags IS 'Array of tags for categorization and filtering';

--bun:split
-- Trigger function to auto-update timestamps
CREATE OR REPLACE FUNCTION notifications_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS notifications_update_trigger ON notifications;

--bun:split
CREATE TRIGGER notifications_update_trigger
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications_update_timestamps();

--bun:split
-- Performance optimization: Set statistics for frequently queried columns
ALTER TABLE notifications
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE notifications
    ALTER COLUMN target_user_id SET STATISTICS 1000;

ALTER TABLE notifications
    ALTER COLUMN event_type SET STATISTICS 500;

