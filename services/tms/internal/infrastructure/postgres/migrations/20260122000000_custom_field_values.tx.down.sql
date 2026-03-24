DROP TRIGGER IF EXISTS custom_field_values_update_timestamp_trigger ON custom_field_values;

--bun:split
DROP FUNCTION IF EXISTS custom_field_values_update_timestamp();

--bun:split
DROP TABLE IF EXISTS "custom_field_values";
