--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

SET statement_timeout = 0;

--bun:split
-- Create formula template category enum
CREATE TYPE formula_template_category_enum AS ENUM(
    'BaseRate',
    'DistanceBased',
    'WeightBased',
    'DimensionalWeight',
    'FuelSurcharge',
    'Accessorial',
    'TimeBasedRate',
    'ZoneBased',
    'Custom'
);

--bun:split
-- Create formula templates table
CREATE TABLE IF NOT EXISTS "formula_templates"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "name" varchar(255) NOT NULL,
    "description" text,
    "category" formula_template_category_enum NOT NULL,
    "expression" text NOT NULL,
    -- Template components (JSONB for flexibility)
    "variables" jsonb,
    "parameters" jsonb,
    -- Metadata
    "tags" text[],
    "examples" jsonb,
    "requirements" jsonb,
    -- Rate constraints
    "min_rate" numeric(19, 4) CHECK ("min_rate" >= 0),
    "max_rate" numeric(19, 4) CHECK ("max_rate" >= 0),
    "output_unit" varchar(50) DEFAULT 'USD',
    -- Status flags
    "is_active" boolean NOT NULL DEFAULT TRUE,
    "is_default" boolean NOT NULL DEFAULT FALSE,
    -- Versioning and timestamps
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_formula_templates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_formula_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_formula_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Business constraints
    CONSTRAINT "chk_formula_templates_expression_length" CHECK (length("expression") > 0),
    CONSTRAINT "chk_formula_templates_name_length" CHECK (length("name") >= 3),
    CONSTRAINT "chk_formula_templates_min_max_rate" CHECK (("min_rate" IS NULL AND "max_rate" IS NULL) OR ("min_rate" IS NOT NULL AND "max_rate" IS NULL) OR ("min_rate" IS NULL AND "max_rate" IS NOT NULL) OR ("min_rate" <= "max_rate"))
);

--bun:split
-- Create unique index for template names within organization
CREATE UNIQUE INDEX "idx_formula_templates_name_unique" ON "formula_templates"(lower("name"), "organization_id");

--bun:split
-- Create unique index to ensure only one default template per category per organization
CREATE UNIQUE INDEX "idx_formula_templates_default_per_category" ON "formula_templates"("organization_id", "category")
WHERE
    "is_default" = TRUE;

--bun:split
-- Performance indexes
CREATE INDEX "idx_formula_templates_organization" ON "formula_templates"("organization_id", "is_active", "category");

CREATE INDEX "idx_formula_templates_business_unit" ON "formula_templates"("business_unit_id");

CREATE INDEX "idx_formula_templates_category" ON "formula_templates"("category", "is_active")
WHERE
    "is_active" = TRUE;

CREATE INDEX "idx_formula_templates_created_updated" ON "formula_templates"("created_at", "updated_at");

--bun:split
-- Create index for tag searching
CREATE INDEX "idx_formula_templates_tags" ON "formula_templates" USING gin("tags")
WHERE
    "tags" IS NOT NULL;

--bun:split
-- Table comments
COMMENT ON TABLE "formula_templates" IS 'Stores formula templates for shipment rate calculations';

--bun:split
-- Column comments
COMMENT ON COLUMN "formula_templates"."id" IS 'Unique identifier for the formula template (pulid with fmt_ prefix)';

COMMENT ON COLUMN "formula_templates"."business_unit_id" IS 'Reference to the business unit that owns this template';

COMMENT ON COLUMN "formula_templates"."organization_id" IS 'Reference to the organization that owns this template';

COMMENT ON COLUMN "formula_templates"."name" IS 'Human-readable name for the template';

COMMENT ON COLUMN "formula_templates"."description" IS 'Detailed description of what the template calculates';

COMMENT ON COLUMN "formula_templates"."category" IS 'Type of rate calculation (e.g., BaseRate, DistanceBased, WeightBased)';

COMMENT ON COLUMN "formula_templates"."expression" IS 'The formula expression used to calculate rates';

COMMENT ON COLUMN "formula_templates"."variables" IS 'Variables used in the formula with their metadata';

COMMENT ON COLUMN "formula_templates"."parameters" IS 'Configurable parameters that can be adjusted by users';

COMMENT ON COLUMN "formula_templates"."tags" IS 'Array of tags for categorization and search';

COMMENT ON COLUMN "formula_templates"."examples" IS 'Example calculations showing how to use the template';

COMMENT ON COLUMN "formula_templates"."requirements" IS 'Requirements that must be met to use this template';

COMMENT ON COLUMN "formula_templates"."min_rate" IS 'Minimum allowed rate to prevent calculation errors';

COMMENT ON COLUMN "formula_templates"."max_rate" IS 'Maximum allowed rate to prevent excessive charges';

COMMENT ON COLUMN "formula_templates"."output_unit" IS 'Currency or unit of the calculated rate (default USD)';

COMMENT ON COLUMN "formula_templates"."is_active" IS 'Whether this template is currently active and usable';

COMMENT ON COLUMN "formula_templates"."is_default" IS 'Whether this template is the default for its category';

COMMENT ON COLUMN "formula_templates"."version" IS 'Version number for optimistic locking';

COMMENT ON COLUMN "formula_templates"."created_at" IS 'Unix timestamp when the template was created';

COMMENT ON COLUMN "formula_templates"."updated_at" IS 'Unix timestamp when the template was last updated';

--bun:split
-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION formula_templates_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS formula_templates_update_trigger ON formula_templates;

--bun:split
CREATE TRIGGER formula_templates_update_trigger
    BEFORE UPDATE ON formula_templates
    FOR EACH ROW
    EXECUTE FUNCTION formula_templates_update_timestamps();

--bun:split
-- Performance optimization
ALTER TABLE "formula_templates"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "formula_templates"
    ALTER COLUMN "category" SET STATISTICS 500;

ALTER TABLE "formula_templates"
    ALTER COLUMN "name" SET STATISTICS 500;

--bun:split
-- Add formula_template_id column to shipments table to link to the template
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100);

--bun:split
-- Add foreign key constraint
ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_formula_template" FOREIGN KEY ("formula_template_id", "business_unit_id", "organization_id") REFERENCES "formula_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- Create index for formula template lookups
CREATE INDEX IF NOT EXISTS "idx_shipments_formula_template" ON "shipments"("formula_template_id", "business_unit_id", "organization_id")
WHERE
    "formula_template_id" IS NOT NULL;

--bun:split
-- Add comment
COMMENT ON COLUMN "shipments"."formula_template_id" IS 'Reference to the formula template used when rating_method is FormulaTemplate';

--bun:split
-- Add check constraint to ensure formula_template_id is set when rating method is FormulaTemplate
ALTER TABLE "shipments"
    ADD CONSTRAINT "chk_shipments_formula_template_required" CHECK (("rating_method" != 'FormulaTemplate') OR ("rating_method" = 'FormulaTemplate' AND "formula_template_id" IS NOT NULL));

