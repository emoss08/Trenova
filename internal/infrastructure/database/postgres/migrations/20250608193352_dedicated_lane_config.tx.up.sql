--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "pattern_configs"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Configuration fields
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "min_frequency" int NOT NULL DEFAULT 3,
    "analysis_window_days" int NOT NULL DEFAULT 90,
    "min_confidence_score" numeric(5, 4) NOT NULL DEFAULT 0.7,
    "suggestion_ttl_days" int NOT NULL DEFAULT 30,
    "require_exact_match" boolean NOT NULL DEFAULT FALSE,
    "weight_recent_shipments" boolean NOT NULL DEFAULT TRUE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_pattern_configs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_pattern_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pattern_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Business logic constraints
    CONSTRAINT "chk_pattern_configs_min_frequency" CHECK ("min_frequency" >= 1 AND "min_frequency" <= 100),
    CONSTRAINT "chk_pattern_configs_analysis_window_days" CHECK ("analysis_window_days" >= 1 AND "analysis_window_days" <= 365),
    CONSTRAINT "chk_pattern_configs_min_confidence_score" CHECK ("min_confidence_score" >= 0.0 AND "min_confidence_score" <= 1.0),
    CONSTRAINT "chk_pattern_configs_suggestion_ttl_days" CHECK ("suggestion_ttl_days" >= 1 AND "suggestion_ttl_days" <= 365)
);

--bun:split
-- Ensure only one configuration per organization
CREATE UNIQUE INDEX IF NOT EXISTS "uq_pattern_configs_organization" ON "pattern_configs"("organization_id");

--bun:split
-- Index for business unit and organization lookups
CREATE INDEX IF NOT EXISTS "idx_pattern_configs_business_unit" ON "pattern_configs"("business_unit_id", "organization_id");

--bun:split
-- Index for tracking creation and updates
CREATE INDEX IF NOT EXISTS "idx_pattern_configs_created_at" ON "pattern_configs"("created_at", "updated_at");

--bun:split
-- Comments for documentation
COMMENT ON TABLE pattern_configs IS 'Stores pattern detection configuration settings for dedicated lane suggestions per organization';

COMMENT ON COLUMN pattern_configs.min_frequency IS 'Minimum number of shipments required to trigger a pattern detection (1-100)';

COMMENT ON COLUMN pattern_configs.analysis_window_days IS 'Number of days to look back when analyzing shipping patterns (1-365)';

COMMENT ON COLUMN pattern_configs.min_confidence_score IS 'Minimum confidence score required to create a dedicated lane suggestion (0.0-1.0)';

COMMENT ON COLUMN pattern_configs.suggestion_ttl_days IS 'Number of days before suggestions expire and are automatically removed (1-365)';

COMMENT ON COLUMN pattern_configs.require_exact_match IS 'Whether pattern detection requires exact matches for equipment and service types';

COMMENT ON COLUMN pattern_configs.weight_recent_shipments IS 'Whether to give more weight to recent shipments in pattern analysis';

--bun:split
-- Trigger function to auto-update timestamps
CREATE OR REPLACE FUNCTION pattern_configs_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS pattern_configs_update_trigger ON pattern_configs;

--bun:split
CREATE TRIGGER pattern_configs_update_trigger
    BEFORE UPDATE ON pattern_configs
    FOR EACH ROW
    EXECUTE FUNCTION pattern_configs_update_timestamps();

--bun:split
-- Performance optimization: Set statistics for frequently queried columns
ALTER TABLE pattern_configs
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE pattern_configs
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

