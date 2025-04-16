-- Remove indexes
DROP INDEX IF EXISTS "idx_integrations_business_unit";
DROP INDEX IF EXISTS "idx_integrations_type";
DROP INDEX IF EXISTS "idx_integrations_config_id";
DROP INDEX IF EXISTS "idx_integrations_created_at";

-- Remove triggers
DROP TRIGGER IF EXISTS "integrations_update_timestamp_trigger" ON "integrations";
DROP FUNCTION IF EXISTS "integrations_update_timestamp";

-- Remove table
DROP TABLE IF EXISTS "integrations"; 