--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

SET statement_timeout = 0;

--bun:split
-- Drop triggers first
DROP TRIGGER IF EXISTS variables_update_trigger ON variables;

--bun:split
-- Drop the trigger function
DROP FUNCTION IF EXISTS variables_update_timestamps();

--bun:split
-- Drop the variables table (cascades to all foreign keys)
DROP TABLE IF EXISTS "variables";

--bun:split
-- Drop the variable_formats table
DROP TABLE IF EXISTS "variable_formats";

--bun:split
-- Drop the custom types
DROP TYPE IF EXISTS variable_value_type_enum;
DROP TYPE IF EXISTS variable_context_enum;
