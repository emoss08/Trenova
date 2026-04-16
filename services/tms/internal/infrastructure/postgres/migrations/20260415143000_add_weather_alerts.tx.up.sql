CREATE TABLE IF NOT EXISTS weather_alerts(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    nws_id TEXT NOT NULL,
    event TEXT NOT NULL,
    severity VARCHAR(50),
    urgency VARCHAR(50),
    certainty VARCHAR(50),
    headline TEXT,
    description TEXT,
    instruction TEXT,
    area_desc TEXT,
    effective BIGINT,
    expires BIGINT,
    onset BIGINT,
    ends BIGINT,
    status VARCHAR(50),
    message_type VARCHAR(50),
    sender_name TEXT,
    response VARCHAR(50),
    category VARCHAR(50),
    alert_category VARCHAR(50) NOT NULL,
    geometry GEOMETRY(Geometry, 4326) NOT NULL,
    first_seen_at BIGINT NOT NULL,
    last_updated_at BIGINT NOT NULL,
    expired_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_weather_alerts PRIMARY KEY (id, organization_id, business_unit_id)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS idx_weather_alerts_tenant_nws_id ON weather_alerts(organization_id, business_unit_id, nws_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_weather_alerts_active ON weather_alerts(organization_id, business_unit_id, expired_at, expires);

--bun:split
CREATE INDEX IF NOT EXISTS idx_weather_alerts_expiration ON weather_alerts(expires, expired_at);

--bun:split
CREATE INDEX IF NOT EXISTS idx_weather_alerts_geometry_gist ON weather_alerts USING GIST (geometry);

--bun:split
CREATE TABLE IF NOT EXISTS weather_alert_activities(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    weather_alert_id VARCHAR(100) NOT NULL,
    activity_type VARCHAR(50) NOT NULL,
    timestamp BIGINT NOT NULL,
    details JSONB,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_weather_alert_activities PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_weather_alert_activities_alert FOREIGN KEY (weather_alert_id, organization_id, business_unit_id) REFERENCES weather_alerts(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_weather_alert_activities_lookup ON weather_alert_activities(organization_id, business_unit_id, weather_alert_id, timestamp DESC);
