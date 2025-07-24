--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Create notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences
(
    id                        varchar(100) PRIMARY KEY,
    user_id                   varchar(100)  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    organization_id           varchar(100)  NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    business_unit_id          varchar(100)  NOT NULL REFERENCES business_units (id) ON DELETE CASCADE,
    -- Configuration
    resource                  varchar(50)   NOT NULL,
    update_types              text[]        NOT NULL DEFAULT '{}',
    notify_on_all_updates     boolean       NOT NULL DEFAULT FALSE,
    notify_only_owned_records boolean       NOT NULL DEFAULT TRUE,
    -- Filtering
    excluded_user_ids         varchar(100)[]         DEFAULT '{}',
    included_role_ids         varchar(100)[]         DEFAULT '{}',
    -- Channel preferences
    preferred_channels        varchar(20)[] NOT NULL,
    -- Timing
    quiet_hours_enabled       boolean       NOT NULL DEFAULT FALSE,
    quiet_hours_start         time,
    quiet_hours_end           time,
    timezone                  varchar(50)   NOT NULL DEFAULT 'UTC',
    -- Batching
    batch_notifications       boolean       NOT NULL DEFAULT FALSE,
    batch_interval_minutes    int           NOT NULL DEFAULT 15,
    -- Status
    is_active                 boolean       NOT NULL DEFAULT TRUE,
    version                   bigint        NOT NULL DEFAULT 0,
    created_at                bigint        NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    updated_at                bigint        NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT chk_batch_interval CHECK (batch_interval_minutes >= 1 AND batch_interval_minutes <= 1440),
    CONSTRAINT chk_quiet_hours CHECK ((quiet_hours_enabled = FALSE) OR
                                      (quiet_hours_enabled = TRUE AND quiet_hours_start IS NOT NULL AND
                                       quiet_hours_end IS NOT NULL))
);

-- Create indexes for notification preferences
CREATE INDEX idx_notification_preferences_user_id ON notification_preferences (user_id);

CREATE INDEX idx_notification_preferences_organization_id ON notification_preferences (organization_id);

CREATE INDEX idx_notification_preferences_resource ON notification_preferences (resource);

CREATE INDEX idx_notification_preferences_active ON notification_preferences (is_active)
    WHERE
        is_active = TRUE;

CREATE UNIQUE INDEX idx_notification_preferences_unique_user_resource ON notification_preferences (user_id, resource)
    WHERE
        is_active = TRUE;

-- Create notification history table
CREATE TABLE IF NOT EXISTS notification_history
(
    id               varchar(100) PRIMARY KEY,
    notification_id  varchar(100) NOT NULL,
    user_id          varchar(100) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    organization_id  varchar(100) NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    business_unit_id varchar(100) NOT NULL REFERENCES business_units (id) ON DELETE CASCADE,
    entity_type      varchar(50),
    entity_id        varchar(100),
    update_type      varchar(50),
    updated_by_id    varchar(100),
    title            varchar(255) NOT NULL,
    message          text         NOT NULL,
    priority         varchar(20)  NOT NULL,
    channel          varchar(20)  NOT NULL,
    event_type       varchar(50)  NOT NULL,
    data             jsonb                 DEFAULT '{}',
    delivery_status  varchar(20)  NOT NULL DEFAULT 'pending',
    delivered_at     bigint,
    failure_reason   text,
    retry_count      int          NOT NULL DEFAULT 0,
    read_at          bigint,
    dismissed_at     bigint,
    clicked_at       bigint,
    action_taken     varchar(100),
    group_id         varchar(100),
    group_position   int,
    created_at       bigint       NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    expires_at       bigint
);

-- Create indexes for notification history
CREATE INDEX idx_notification_history_user_id ON notification_history (user_id);

CREATE INDEX idx_notification_history_organization_id ON notification_history (organization_id);

CREATE INDEX idx_notification_history_notification_id ON notification_history (notification_id);

CREATE INDEX idx_notification_history_created_at ON notification_history (created_at DESC);

CREATE INDEX idx_notification_history_unread ON notification_history (user_id, read_at)
    WHERE
        read_at IS NULL;

CREATE INDEX idx_notification_history_entity ON notification_history (entity_type, entity_id);

CREATE INDEX idx_notification_history_group ON notification_history (group_id, group_position)
    WHERE
        group_id IS NOT NULL;

-- Create notification rate limits table
CREATE TABLE IF NOT EXISTS notification_rate_limits
(
    id                 varchar(100) PRIMARY KEY,
    organization_id    varchar(100) NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    -- Rule configuration
    name               varchar(100) NOT NULL,
    description        text,
    resource           varchar(50),
    event_type         varchar(50),
    priority           varchar(20),
    -- Rate limit settings
    max_notifications  int          NOT NULL,
    period             varchar(20)  NOT NULL,
    -- Scope
    apply_to_all_users boolean      NOT NULL DEFAULT TRUE,
    user_id            varchar(100) REFERENCES users (id) ON DELETE CASCADE,
    role_id            varchar(100),
    -- Status
    is_active          boolean      NOT NULL DEFAULT TRUE,
    version            bigint       NOT NULL DEFAULT 0,
    created_at         bigint       NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    updated_at         bigint       NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT chk_max_notifications CHECK (max_notifications >= 1),
    CONSTRAINT chk_period CHECK (period IN ('minute', 'hour', 'day')),
    CONSTRAINT chk_scope CHECK (apply_to_all_users = TRUE OR user_id IS NOT NULL OR role_id IS NOT NULL)
);

-- Create indexes for notification rate limits
CREATE INDEX idx_notification_rate_limits_organization_id ON notification_rate_limits (organization_id);

CREATE INDEX idx_notification_rate_limits_active ON notification_rate_limits (is_active)
    WHERE
        is_active = TRUE;

CREATE INDEX idx_notification_rate_limits_user_id ON notification_rate_limits (user_id)
    WHERE
        user_id IS NOT NULL;

CREATE INDEX idx_notification_rate_limits_resource ON notification_rate_limits (resource)
    WHERE
        resource IS NOT NULL;

