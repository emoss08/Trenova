-- Create notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id VARCHAR(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    
    -- Configuration
    resource VARCHAR(50) NOT NULL,
    update_types TEXT[] NOT NULL DEFAULT '{}',
    notify_on_all_updates BOOLEAN NOT NULL DEFAULT false,
    notify_only_owned_records BOOLEAN NOT NULL DEFAULT true,
    
    -- Filtering
    excluded_user_ids VARCHAR(100)[] DEFAULT '{}',
    included_role_ids VARCHAR(100)[] DEFAULT '{}',
    
    -- Channel preferences
    preferred_channels VARCHAR(20)[] NOT NULL,
    
    -- Timing
    quiet_hours_enabled BOOLEAN NOT NULL DEFAULT false,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    
    -- Batching
    batch_notifications BOOLEAN NOT NULL DEFAULT false,
    batch_interval_minutes INT NOT NULL DEFAULT 15,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    
    CONSTRAINT chk_batch_interval CHECK (batch_interval_minutes >= 1 AND batch_interval_minutes <= 1440),
    CONSTRAINT chk_quiet_hours CHECK ((quiet_hours_enabled = false) OR (quiet_hours_enabled = true AND quiet_hours_start IS NOT NULL AND quiet_hours_end IS NOT NULL))
);

-- Create indexes for notification preferences
CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);
CREATE INDEX idx_notification_preferences_organization_id ON notification_preferences(organization_id);
CREATE INDEX idx_notification_preferences_resource ON notification_preferences(resource);
CREATE INDEX idx_notification_preferences_active ON notification_preferences(is_active) WHERE is_active = true;
CREATE UNIQUE INDEX idx_notification_preferences_unique_user_resource ON notification_preferences(user_id, resource) WHERE is_active = true;

-- Create notification history table
CREATE TABLE IF NOT EXISTS notification_history (
    id VARCHAR(100) PRIMARY KEY,
    notification_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    business_unit_id VARCHAR(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    
    -- Entity reference
    entity_type VARCHAR(50),
    entity_id VARCHAR(100),
    update_type VARCHAR(50),
    updated_by_id VARCHAR(100),
    
    -- Notification details
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL,
    channel VARCHAR(20) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    data JSONB DEFAULT '{}',
    
    -- Delivery information
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    delivered_at BIGINT,
    failure_reason TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    
    -- User interaction
    read_at BIGINT,
    dismissed_at BIGINT,
    clicked_at BIGINT,
    action_taken VARCHAR(100),
    
    -- Grouping
    group_id VARCHAR(100),
    group_position INT,
    
    -- Timestamps
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    expires_at BIGINT
);

-- Create indexes for notification history
CREATE INDEX idx_notification_history_user_id ON notification_history(user_id);
CREATE INDEX idx_notification_history_organization_id ON notification_history(organization_id);
CREATE INDEX idx_notification_history_notification_id ON notification_history(notification_id);
CREATE INDEX idx_notification_history_created_at ON notification_history(created_at DESC);
CREATE INDEX idx_notification_history_unread ON notification_history(user_id, read_at) WHERE read_at IS NULL;
CREATE INDEX idx_notification_history_entity ON notification_history(entity_type, entity_id);
CREATE INDEX idx_notification_history_group ON notification_history(group_id, group_position) WHERE group_id IS NOT NULL;

-- Create notification rate limits table
CREATE TABLE IF NOT EXISTS notification_rate_limits (
    id VARCHAR(100) PRIMARY KEY,
    organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    
    -- Rule configuration
    name VARCHAR(100) NOT NULL,
    description TEXT,
    resource VARCHAR(50),
    event_type VARCHAR(50),
    priority VARCHAR(20),
    
    -- Rate limit settings
    max_notifications INT NOT NULL,
    period VARCHAR(20) NOT NULL,
    
    -- Scope
    apply_to_all_users BOOLEAN NOT NULL DEFAULT true,
    user_id VARCHAR(100) REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(100),
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    
    CONSTRAINT chk_max_notifications CHECK (max_notifications >= 1),
    CONSTRAINT chk_period CHECK (period IN ('minute', 'hour', 'day')),
    CONSTRAINT chk_scope CHECK (apply_to_all_users = true OR user_id IS NOT NULL OR role_id IS NOT NULL)
);

-- Create indexes for notification rate limits
CREATE INDEX idx_notification_rate_limits_organization_id ON notification_rate_limits(organization_id);
CREATE INDEX idx_notification_rate_limits_active ON notification_rate_limits(is_active) WHERE is_active = true;
CREATE INDEX idx_notification_rate_limits_user_id ON notification_rate_limits(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_notification_rate_limits_resource ON notification_rate_limits(resource) WHERE resource IS NOT NULL;