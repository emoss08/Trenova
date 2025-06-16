--changeset manu:add_notification_preferences_table

-- Create notification preferences table
CREATE TABLE IF NOT EXISTS "notification_preferences"(
    "id" VARCHAR(100) PRIMARY KEY,
    "user_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    
    -- Configuration
    "resource" VARCHAR(50) NOT NULL,
    "update_types" TEXT[] NOT NULL DEFAULT '{}',
    "notify_on_all_updates" BOOLEAN NOT NULL DEFAULT FALSE,
    "notify_only_owned_records" BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Filtering
    "excluded_user_ids" VARCHAR(100)[] DEFAULT '{}',
    "included_role_ids" VARCHAR(100)[] DEFAULT '{}',
    
    -- Channel preferences
    "preferred_channels" VARCHAR(20)[] NOT NULL DEFAULT '{user}',
    
    -- Timing
    "quiet_hours_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    "quiet_hours_start" TIME,
    "quiet_hours_end" TIME,
    "timezone" VARCHAR(50) NOT NULL DEFAULT 'UTC',
    
    -- Batching
    "batch_notifications" BOOLEAN NOT NULL DEFAULT FALSE,
    "batch_interval_minutes" INT NOT NULL DEFAULT 15,
    
    -- Status
    "is_active" BOOLEAN NOT NULL DEFAULT TRUE,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    
    -- Constraints
    CONSTRAINT "fk_notification_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notification_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notification_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_batch_interval" CHECK ("batch_interval_minutes" >= 1 AND "batch_interval_minutes" <= 1440)
);

--bun:split

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS "idx_notification_preferences_user" ON "notification_preferences"("user_id");
CREATE INDEX IF NOT EXISTS "idx_notification_preferences_org" ON "notification_preferences"("organization_id");
CREATE INDEX IF NOT EXISTS "idx_notification_preferences_resource" ON "notification_preferences"("resource");
CREATE INDEX IF NOT EXISTS "idx_notification_preferences_active" ON "notification_preferences"("is_active") WHERE "is_active" = TRUE;

--bun:split

-- Create composite index for common queries
CREATE INDEX IF NOT EXISTS "idx_notification_preferences_user_resource_active" 
ON "notification_preferences"("user_id", "resource", "is_active") 
WHERE "is_active" = TRUE;

--bun:split

-- Add comments
COMMENT ON TABLE "notification_preferences" IS 'Stores user preferences for notifications on owned record updates';
COMMENT ON COLUMN "notification_preferences"."resource" IS 'Type of resource (shipment, worker, customer, etc.)';
COMMENT ON COLUMN "notification_preferences"."update_types" IS 'Types of updates to notify about (status_change, assignment, etc.)';
COMMENT ON COLUMN "notification_preferences"."notify_on_all_updates" IS 'Whether to notify on all update types';
COMMENT ON COLUMN "notification_preferences"."notify_only_owned_records" IS 'Whether to only notify for records owned/created by the user';
COMMENT ON COLUMN "notification_preferences"."excluded_user_ids" IS 'User IDs to exclude from triggering notifications';
COMMENT ON COLUMN "notification_preferences"."included_role_ids" IS 'Role IDs to include for triggering notifications';
COMMENT ON COLUMN "notification_preferences"."preferred_channels" IS 'Preferred notification channels (user, email, etc.)';
COMMENT ON COLUMN "notification_preferences"."quiet_hours_enabled" IS 'Whether quiet hours are enabled';
COMMENT ON COLUMN "notification_preferences"."quiet_hours_start" IS 'Start time for quiet hours';
COMMENT ON COLUMN "notification_preferences"."quiet_hours_end" IS 'End time for quiet hours';
COMMENT ON COLUMN "notification_preferences"."batch_notifications" IS 'Whether to batch notifications';
COMMENT ON COLUMN "notification_preferences"."batch_interval_minutes" IS 'Interval in minutes for batching notifications';