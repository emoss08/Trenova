SET statement_timeout = 0;

--bun:split
-- Drop triggers first
DROP TRIGGER IF EXISTS formula_schemas_update_trigger ON formula_schemas;
DROP TRIGGER IF EXISTS formula_contexts_update_trigger ON formula_contexts;

--bun:split
-- Drop the trigger function
DROP FUNCTION IF EXISTS formula_contexts_update_timestamps();

--bun:split
-- Drop tables (this will cascade delete all data and foreign key references)
DROP TABLE IF EXISTS "formula_schemas";
DROP TABLE IF EXISTS "formula_contexts";

--bun:split
-- Drop the custom types
DROP TYPE IF EXISTS value_type_enum;
DROP TYPE IF EXISTS context_type_enum;