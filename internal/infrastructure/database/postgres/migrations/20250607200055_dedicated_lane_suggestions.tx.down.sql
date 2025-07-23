-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

-- Drop dedicated lane suggestions table and related objects

--bun:split
-- Drop all indexes
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_dashboard" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_high_value" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_unique_pending_pattern" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_timestamps" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_equipment" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_confidence" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_temporal" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_pattern_details" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_analytics" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_expiration" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_locations" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_customer" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_pending" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_status" CASCADE;

--bun:split
DROP INDEX IF EXISTS "idx_dedicated_lane_suggestions_business_unit_org" CASCADE;

--bun:split
-- Drop trigger and function
DROP TRIGGER IF EXISTS dedicated_lane_suggestions_update_trigger ON dedicated_lane_suggestions CASCADE;

--bun:split
DROP FUNCTION IF EXISTS dedicated_lane_suggestions_update_timestamps() CASCADE;

--bun:split
-- Drop the main table
DROP TABLE IF EXISTS "dedicated_lane_suggestions" CASCADE;

--bun:split
-- Drop the enum type
DROP TYPE IF EXISTS "suggestion_status_enum" CASCADE;