--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Drop triggers first
DROP TRIGGER IF EXISTS pattern_configs_update_trigger ON pattern_configs;

--bun:split
-- Drop trigger function
DROP FUNCTION IF EXISTS pattern_configs_update_timestamps();

--bun:split
-- Drop indexes
DROP INDEX IF EXISTS idx_pattern_configs_organization_unique;

--bun:split
DROP INDEX IF EXISTS idx_pattern_configs_business_unit;

--bun:split
DROP INDEX IF EXISTS idx_pattern_configs_created_at;

--bun:split
-- Drop the table (this will cascade and remove all foreign key references)
DROP TABLE IF EXISTS pattern_configs;
