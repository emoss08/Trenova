DROP INDEX IF EXISTS idx_weather_alert_activities_lookup;

--bun:split
DROP TABLE IF EXISTS weather_alert_activities;

--bun:split
DROP INDEX IF EXISTS idx_weather_alerts_geometry_gist;

--bun:split
DROP INDEX IF EXISTS idx_weather_alerts_expiration;

--bun:split
DROP INDEX IF EXISTS idx_weather_alerts_active;

--bun:split
DROP INDEX IF EXISTS idx_weather_alerts_tenant_nws_id;

--bun:split
DROP TABLE IF EXISTS weather_alerts;
