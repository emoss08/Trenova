DROP TRIGGER IF EXISTS custom_field_definitions_update_timestamp_trigger ON custom_field_definitions;

--bun:split
DROP FUNCTION IF EXISTS custom_field_definitions_update_timestamp();

--bun:split
DROP TABLE IF EXISTS "custom_field_definitions";

--bun:split
DROP TYPE IF EXISTS "custom_field_type_enum";
