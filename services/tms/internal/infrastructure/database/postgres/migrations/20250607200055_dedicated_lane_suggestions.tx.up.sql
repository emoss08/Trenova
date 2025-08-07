--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Dedicated Lane Suggestion Status Enum
CREATE TYPE "suggestion_status_enum" AS ENUM(
    'Pending', -- Suggestion awaiting review
    'Accepted', -- Suggestion approved and dedicated lane created
    'Rejected', -- Suggestion declined by user
    'Expired' -- Suggestion expired due to TTL
);

--bun:split
-- Dedicated Lane Suggestions Table
CREATE TABLE IF NOT EXISTS "dedicated_lane_suggestions"(
    -- Primary Identifiers (Multi-tenant Architecture)
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core Status and Identification
    "status" suggestion_status_enum NOT NULL DEFAULT 'Pending',
    "customer_id" varchar(100) NOT NULL,
    "origin_location_id" varchar(100) NOT NULL,
    "destination_location_id" varchar(100) NOT NULL,
    -- Optional Equipment/Service Configuration
    "service_type_id" varchar(100),
    "shipment_type_id" varchar(100),
    "trailer_type_id" varchar(100),
    "tractor_type_id" varchar(100),
    -- Pattern Analysis Metrics
    "confidence_score" numeric(5, 4) NOT NULL,
    "frequency_count" integer NOT NULL,
    "average_freight_charge" numeric(19, 4),
    "total_freight_value" numeric(19, 4),
    -- Temporal Pattern Information
    "last_shipment_date" bigint NOT NULL,
    "first_shipment_date" bigint NOT NULL,
    -- Suggestion Management
    "suggested_name" varchar(200) NOT NULL,
    "pattern_details" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    "created_dedicated_lane_id" varchar(100),
    "processed_by_id" varchar(100),
    "processed_at" bigint,
    "expires_at" bigint NOT NULL,
    -- Standard Metadata and Versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Primary Key and Multi-tenant Constraints
    CONSTRAINT "pk_dedicated_lane_suggestions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    -- Foreign Key Constraints (Multi-tenant)
    CONSTRAINT "fk_dedicated_lane_suggestions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lane_suggestions_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    -- Optional Equipment/Service Foreign Keys
    CONSTRAINT "fk_dedicated_lane_suggestions_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_trailer_type" FOREIGN KEY ("trailer_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_tractor_type" FOREIGN KEY ("tractor_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_created_lane" FOREIGN KEY ("created_dedicated_lane_id", "business_unit_id", "organization_id") REFERENCES "dedicated_lanes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_processed_by" FOREIGN KEY ("processed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    -- Business Logic Constraints
    CONSTRAINT "chk_dedicated_lane_suggestions_different_locations" CHECK (origin_location_id != destination_location_id),
    CONSTRAINT "chk_dedicated_lane_suggestions_confidence_score" CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    CONSTRAINT "chk_dedicated_lane_suggestions_frequency_count" CHECK (frequency_count >= 1),
    CONSTRAINT "chk_dedicated_lane_suggestions_pattern_details_format" CHECK (jsonb_typeof(pattern_details) = 'object'),
    -- Processing Logic Constraints
    CONSTRAINT "chk_dedicated_lane_suggestions_processed_logic" CHECK ((status = 'Pending' AND processed_by_id IS NULL AND processed_at IS NULL) OR (status IN ('Accepted', 'Rejected') AND processed_by_id IS NOT NULL AND processed_at IS NOT NULL) OR (status = 'Expired')),
    CONSTRAINT "chk_dedicated_lane_suggestions_acceptance_logic" CHECK ((status = 'Accepted' AND created_dedicated_lane_id IS NOT NULL) OR (status != 'Accepted' AND created_dedicated_lane_id IS NULL))
);

--bun:split
-- Primary Business Unit/Organization Index (Most Common Query Pattern)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_business_unit_org" ON "dedicated_lane_suggestions"("business_unit_id", "organization_id");

--bun:split
-- Status Filtering Index (Critical for Dashboards and Processing)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_status" ON "dedicated_lane_suggestions"("status", "organization_id") INCLUDE ("business_unit_id", "customer_id", "confidence_score");

--bun:split
-- Pending Suggestions Index (Primary Active Workflow)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_pending" ON "dedicated_lane_suggestions"("organization_id", "business_unit_id", "confidence_score" DESC, "created_at" DESC)
WHERE
    status = 'Pending';

--bun:split
-- Customer Analysis Index (Customer-Specific Reports)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_customer" ON "dedicated_lane_suggestions"("customer_id", "business_unit_id", "organization_id", "status", "confidence_score" DESC);

--bun:split
-- Location Pattern Index (Route Analysis)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_locations" ON "dedicated_lane_suggestions"("origin_location_id", "destination_location_id", "customer_id");

--bun:split
-- Expiration Management Index (Background Processing)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_expiration" ON "dedicated_lane_suggestions"("expires_at", "status")
WHERE
    status = 'Pending';

--bun:split
-- Analytics and Reporting Index (Performance Metrics)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_analytics" ON "dedicated_lane_suggestions"("processed_at", "status", "confidence_score")
WHERE
    processed_at IS NOT NULL;

--bun:split
-- Pattern Details JSONB Index (Advanced Search and Analytics)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_pattern_details" ON "dedicated_lane_suggestions" USING gin("pattern_details");

--bun:split
-- Confidence Score Performance Index (High-Value Suggestions)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_confidence" ON "dedicated_lane_suggestions"("confidence_score" DESC, "frequency_count" DESC)
WHERE
    status = 'Pending' AND confidence_score >= 0.7;

--bun:split
-- Equipment Type Analysis Index (Equipment-Specific Patterns)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_equipment" ON "dedicated_lane_suggestions"("tractor_type_id", "trailer_type_id", "service_type_id", "shipment_type_id")
WHERE
    tractor_type_id IS NOT NULL OR trailer_type_id IS NOT NULL;

--bun:split
-- Created-Updated Timestamp Index (Standard Audit Trail)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_timestamps" ON "dedicated_lane_suggestions"("created_at" DESC, "updated_at" DESC);

--bun:split
-- Comprehensive Comments for Enterprise Documentation
COMMENT ON TABLE dedicated_lane_suggestions IS 'AI-driven suggestions for creating dedicated lanes based on recurring shipment patterns. Supports pattern detection, confidence scoring, and automated suggestion workflows for transportation optimization.';

COMMENT ON COLUMN dedicated_lane_suggestions.status IS 'Current processing status of the suggestion (Pending, Accepted, Rejected, Expired)';

COMMENT ON COLUMN dedicated_lane_suggestions.customer_id IS 'Customer for whom this dedicated lane is suggested based on their shipping patterns';

COMMENT ON COLUMN dedicated_lane_suggestions.origin_location_id IS 'Frequently used origin location identified in the shipping pattern';

COMMENT ON COLUMN dedicated_lane_suggestions.destination_location_id IS 'Frequently used destination location identified in the shipping pattern';

COMMENT ON COLUMN dedicated_lane_suggestions.confidence_score IS 'Algorithm-calculated confidence score (0.0-1.0) indicating pattern strength and suggestion reliability';

COMMENT ON COLUMN dedicated_lane_suggestions.frequency_count IS 'Number of shipments found matching this pattern during analysis period';

COMMENT ON COLUMN dedicated_lane_suggestions.average_freight_charge IS 'Average freight charge across all shipments in this pattern for ROI analysis';

COMMENT ON COLUMN dedicated_lane_suggestions.total_freight_value IS 'Total freight value across all pattern shipments for business impact assessment';

COMMENT ON COLUMN dedicated_lane_suggestions.last_shipment_date IS 'Most recent shipment date in the detected pattern (Unix timestamp)';

COMMENT ON COLUMN dedicated_lane_suggestions.first_shipment_date IS 'Earliest shipment date in the detected pattern (Unix timestamp)';

COMMENT ON COLUMN dedicated_lane_suggestions.suggested_name IS 'Algorithm-generated suggested name for the dedicated lane';

COMMENT ON COLUMN dedicated_lane_suggestions.pattern_details IS 'JSONB containing detailed pattern analysis data, shipment IDs, and algorithm metadata';

COMMENT ON COLUMN dedicated_lane_suggestions.created_dedicated_lane_id IS 'Reference to the dedicated lane created when this suggestion was accepted';

COMMENT ON COLUMN dedicated_lane_suggestions.processed_by_id IS 'User who processed (accepted/rejected) this suggestion';

COMMENT ON COLUMN dedicated_lane_suggestions.processed_at IS 'Timestamp when the suggestion was processed';

COMMENT ON COLUMN dedicated_lane_suggestions.expires_at IS 'Expiration timestamp for this suggestion based on configured TTL';

--bun:split
-- Auto-update Timestamp Trigger Function
CREATE OR REPLACE FUNCTION dedicated_lane_suggestions_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Auto-update timestamps on modification
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
-- Drop Existing Trigger (Safe)
DROP TRIGGER IF EXISTS dedicated_lane_suggestions_update_trigger ON dedicated_lane_suggestions;

--bun:split
-- Create Update Trigger
CREATE TRIGGER dedicated_lane_suggestions_update_trigger
    BEFORE UPDATE ON dedicated_lane_suggestions
    FOR EACH ROW
    EXECUTE FUNCTION dedicated_lane_suggestions_update_timestamps();

--bun:split
-- Performance Optimization: Set Statistics for Query Planner
ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN customer_id SET STATISTICS 1000;

ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN status SET STATISTICS 1000;

ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN confidence_score SET STATISTICS 1000;

ALTER TABLE dedicated_lane_suggestions
    ALTER COLUMN expires_at SET STATISTICS 1000;

--bun:split
-- Unique Constraint to Prevent Duplicate Active Suggestions
-- This prevents multiple pending suggestions for the same pattern
CREATE UNIQUE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_unique_pending_pattern" ON "dedicated_lane_suggestions"("customer_id", "origin_location_id", "destination_location_id", "organization_id", COALESCE("service_type_id", ''), COALESCE("shipment_type_id", ''), COALESCE("trailer_type_id", ''), COALESCE("tractor_type_id", ''))
WHERE
    status = 'Pending';

--bun:split
-- Partial Index for High-Value Suggestions (Enterprise Optimization)
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_high_value" ON "dedicated_lane_suggestions"("total_freight_value" DESC, "frequency_count" DESC, "confidence_score" DESC)
WHERE
    status = 'Pending' AND total_freight_value IS NOT NULL AND total_freight_value > 10000;

--bun:split
-- Composite Index for Most Common Dashboard Query
CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_dashboard" ON "dedicated_lane_suggestions"("organization_id", "business_unit_id", "status", "created_at" DESC) INCLUDE ("customer_id", "confidence_score", "frequency_count", "total_freight_value");

