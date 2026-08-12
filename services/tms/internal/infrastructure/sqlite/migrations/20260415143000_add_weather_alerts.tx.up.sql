-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260415143000_add_weather_alerts.tx.up.sql

CREATE TABLE IF NOT EXISTS weather_alerts(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "nws_id" TEXT NOT NULL,
    "event" TEXT NOT NULL,
    "severity" TEXT,
    "urgency" TEXT,
    "certainty" TEXT,
    "headline" TEXT,
    "description" TEXT,
    "instruction" TEXT,
    "area_desc" TEXT,
    "effective" INTEGER,
    "expires" INTEGER,
    "onset" INTEGER,
    "ends" INTEGER,
    "status" TEXT,
    "message_type" TEXT,
    "sender_name" TEXT,
    "response" TEXT,
    "category" TEXT,
    "alert_category" TEXT NOT NULL,
    "first_seen_at" INTEGER NOT NULL,
    "last_updated_at" INTEGER NOT NULL,
    "expired_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_weather_alerts PRIMARY KEY (id, organization_id, business_unit_id)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_weather_alerts_tenant_nws_id ON weather_alerts (organization_id, business_unit_id, nws_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_weather_alerts_active ON weather_alerts (organization_id, business_unit_id, expired_at, expires);

--bun:split

CREATE INDEX IF NOT EXISTS idx_weather_alerts_expiration ON weather_alerts (expires, expired_at);

--bun:split

CREATE TABLE IF NOT EXISTS weather_alert_activities(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "weather_alert_id" TEXT NOT NULL,
    "activity_type" TEXT NOT NULL,
    "timestamp" INTEGER NOT NULL,
    "details" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_weather_alert_activities PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_weather_alert_activities_alert FOREIGN KEY (weather_alert_id, organization_id, business_unit_id) REFERENCES weather_alerts(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_weather_alert_activities_lookup ON weather_alert_activities (organization_id, business_unit_id, weather_alert_id, timestamp DESC);
