CREATE TABLE IF NOT EXISTS user_push_subscriptions(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent VARCHAR(255),
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_user_push_subscriptions PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_push_subscriptions_endpoint ON user_push_subscriptions(md5(endpoint));

--bun:split
CREATE INDEX IF NOT EXISTS idx_user_push_subscriptions_user ON user_push_subscriptions(user_id);
