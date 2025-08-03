--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

SET statement_timeout = 0;

--bun:split
-- Create enum for context types
CREATE TYPE context_type_enum AS ENUM ('BUILT_IN', 'CUSTOM');

--bun:split
-- Create enum for value types
CREATE TYPE value_type_enum AS ENUM ('NUMBER', 'STRING', 'BOOLEAN', 'DATE', 'ARRAY', 'OBJECT');

--bun:split
-- Create the formula_contexts table
CREATE TABLE IF NOT EXISTS "formula_contexts" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    
    -- Core fields
    "name" varchar(100) NOT NULL,
    "context_type" context_type_enum NOT NULL DEFAULT 'CUSTOM',
    "data_source" text,
    "value_type" value_type_enum NOT NULL,
    "validation_rules" jsonb,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    
    -- Constraints
    CONSTRAINT "pk_formula_contexts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_formula_contexts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_formula_contexts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    
    -- Business constraints
    CONSTRAINT "chk_formula_contexts_custom_data_source" CHECK (
        ("context_type" = 'CUSTOM' AND "data_source" IS NOT NULL) OR 
        ("context_type" = 'BUILT_IN')
    ),
    CONSTRAINT "chk_formula_contexts_name_length" CHECK (length("name") > 0)
);

--bun:split
-- Create unique index for context names within organization
CREATE UNIQUE INDEX "idx_formula_contexts_name_unique" 
ON "formula_contexts" (lower("name"), "organization_id")
WHERE "is_system" = FALSE;

--bun:split
-- Create index for organization-based queries
CREATE INDEX "idx_formula_contexts_organization" 
ON "formula_contexts" ("organization_id", "context_type", "name");

--bun:split
-- Create index for business unit queries
CREATE INDEX "idx_formula_contexts_business_unit" 
ON "formula_contexts" ("business_unit_id");

--bun:split
-- Create index for system contexts
CREATE INDEX "idx_formula_contexts_system" 
ON "formula_contexts" ("is_system", "context_type")
WHERE "is_system" = TRUE;

--bun:split
-- Create index for timestamps (for audit/cleanup)
CREATE INDEX "idx_formula_contexts_created_updated" 
ON "formula_contexts" ("created_at", "updated_at");

--bun:split
-- Table comments
COMMENT ON TABLE "formula_contexts" IS 'Stores formula context definitions that provide data access for pricing and calculation formulas';

--bun:split
-- Column comments
COMMENT ON COLUMN "formula_contexts"."id" IS 'Unique identifier for the formula context (pulid with fctx_ prefix)';
COMMENT ON COLUMN "formula_contexts"."business_unit_id" IS 'Reference to the business unit that owns this context';
COMMENT ON COLUMN "formula_contexts"."organization_id" IS 'Reference to the organization that owns this context';
COMMENT ON COLUMN "formula_contexts"."name" IS 'Human-readable name for the context (e.g., "Equipment Cost", "Temperature Range")';
COMMENT ON COLUMN "formula_contexts"."context_type" IS 'Type of context: BUILT_IN (system-provided) or CUSTOM (user-defined)';
COMMENT ON COLUMN "formula_contexts"."data_source" IS 'JSON path, field reference, or computation definition for extracting context value';
COMMENT ON COLUMN "formula_contexts"."value_type" IS 'Expected data type of the context value';
COMMENT ON COLUMN "formula_contexts"."validation_rules" IS 'JSON object containing validation rules for the context value';
COMMENT ON COLUMN "formula_contexts"."is_system" IS 'Whether this is a system-provided context that cannot be modified';
COMMENT ON COLUMN "formula_contexts"."version" IS 'Version number for optimistic locking';
COMMENT ON COLUMN "formula_contexts"."created_at" IS 'Unix timestamp when the context was created';
COMMENT ON COLUMN "formula_contexts"."updated_at" IS 'Unix timestamp when the context was last updated';

--bun:split
-- Create formula_schemas table to store JSON schema definitions
CREATE TABLE IF NOT EXISTS "formula_schemas" (
    -- Primary identifiers
    "id" varchar(255) NOT NULL, -- JSON Schema $id
    "organization_id" varchar(100) NOT NULL,
    
    -- Schema metadata
    "schema_uri" varchar(255) NOT NULL, -- JSON Schema $schema
    "title" varchar(255) NOT NULL,
    "description" text,
    "type" varchar(50) NOT NULL DEFAULT 'object',
    "version" varchar(50) NOT NULL,
    
    -- Schema content
    "properties" jsonb NOT NULL,
    "required" text[],
    "data_source" jsonb, -- x-data-source extension
    "formula_context" jsonb, -- x-formula-context extension
    
    -- Management
    "is_active" boolean NOT NULL DEFAULT TRUE,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    
    -- Constraints
    CONSTRAINT "pk_formula_schemas" PRIMARY KEY ("id", "organization_id"),
    CONSTRAINT "fk_formula_schemas_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_formula_schemas_uri" UNIQUE ("schema_uri", "organization_id")
);

--bun:split
-- Indexes for formula_schemas
CREATE INDEX "idx_formula_schemas_organization" 
ON "formula_schemas" ("organization_id", "is_active");

CREATE INDEX "idx_formula_schemas_system" 
ON "formula_schemas" ("is_system")
WHERE "is_system" = TRUE;

CREATE INDEX "idx_formula_schemas_created_updated" 
ON "formula_schemas" ("created_at", "updated_at");

--bun:split
-- Table and column comments for formula_schemas
COMMENT ON TABLE "formula_schemas" IS 'Stores JSON Schema definitions for formula data structures and validation';

COMMENT ON COLUMN "formula_schemas"."id" IS 'JSON Schema $id field - unique identifier for the schema';
COMMENT ON COLUMN "formula_schemas"."schema_uri" IS 'JSON Schema $schema field - URI of the JSON Schema version';
COMMENT ON COLUMN "formula_schemas"."title" IS 'Human-readable title of the schema';
COMMENT ON COLUMN "formula_schemas"."description" IS 'Detailed description of what this schema represents';
COMMENT ON COLUMN "formula_schemas"."type" IS 'JSON Schema type (typically "object" for root schemas)';
COMMENT ON COLUMN "formula_schemas"."version" IS 'Version of this schema definition';
COMMENT ON COLUMN "formula_schemas"."properties" IS 'JSON Schema properties definition';
COMMENT ON COLUMN "formula_schemas"."required" IS 'Array of required property names';
COMMENT ON COLUMN "formula_schemas"."data_source" IS 'Custom x-data-source extension for data mapping';
COMMENT ON COLUMN "formula_schemas"."formula_context" IS 'Custom x-formula-context extension for formula metadata';
COMMENT ON COLUMN "formula_schemas"."is_active" IS 'Whether this schema is currently active and usable';
COMMENT ON COLUMN "formula_schemas"."is_system" IS 'Whether this is a system-provided schema';

--bun:split
-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION formula_contexts_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS formula_contexts_update_trigger ON formula_contexts;

--bun:split
CREATE TRIGGER formula_contexts_update_trigger
    BEFORE UPDATE ON formula_contexts
    FOR EACH ROW
    EXECUTE FUNCTION formula_contexts_update_timestamps();

--bun:split
DROP TRIGGER IF EXISTS formula_schemas_update_trigger ON formula_schemas;

--bun:split
CREATE TRIGGER formula_schemas_update_trigger
    BEFORE UPDATE ON formula_schemas
    FOR EACH ROW
    EXECUTE FUNCTION formula_contexts_update_timestamps();

--bun:split
-- Insert built-in contexts for shipments
INSERT INTO "formula_contexts" (
    "id", 
    "business_unit_id", 
    "organization_id", 
    "name", 
    "context_type", 
    "data_source", 
    "value_type", 
    "is_system"
)
SELECT 
    'fctx_sys_weight',
    o.business_unit_id,
    o.id,
    'Shipment Weight',
    'BUILT_IN',
    'field:weight',
    'NUMBER',
    TRUE
FROM "organizations" o
ON CONFLICT ("id", "business_unit_id", "organization_id") DO NOTHING;

--bun:split
INSERT INTO "formula_contexts" (
    "id", 
    "business_unit_id", 
    "organization_id", 
    "name", 
    "context_type", 
    "data_source", 
    "value_type", 
    "is_system"
)
SELECT 
    'fctx_sys_temp_diff',
    o.business_unit_id,
    o.id,
    'Temperature Differential',
    'BUILT_IN',
    'compute:temperatureDifferential',
    'NUMBER',
    TRUE
FROM "organizations" o
ON CONFLICT ("id", "business_unit_id", "organization_id") DO NOTHING;

--bun:split
INSERT INTO "formula_contexts" (
    "id", 
    "business_unit_id", 
    "organization_id", 
    "name", 
    "context_type", 
    "data_source", 
    "value_type", 
    "is_system"
)
SELECT 
    'fctx_sys_has_hazmat',
    o.business_unit_id,
    o.id,
    'Has Hazardous Materials',
    'BUILT_IN',
    'compute:hasHazmat',
    'BOOLEAN',
    TRUE
FROM "organizations" o
ON CONFLICT ("id", "business_unit_id", "organization_id") DO NOTHING;

--bun:split
-- Performance optimization
ALTER TABLE "formula_contexts"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "formula_contexts"
    ALTER COLUMN "name" SET STATISTICS 500;

ALTER TABLE "formula_schemas"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;