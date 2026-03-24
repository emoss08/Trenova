--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

SET statement_timeout = 0;

--bun:split
-- Create variable context enum
CREATE TYPE variable_context_enum AS ENUM (
    'Invoice',
    'Customer',
    'Shipment',
    'Organization',
    'System'
);

--bun:split
-- Create variable value type enum
CREATE TYPE variable_value_type_enum AS ENUM (
    'String',
    'Number',
    'Date',
    'Boolean',
    'Currency'
);

--bun:split
-- Create variable formats table
CREATE TABLE IF NOT EXISTS "variable_formats"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,

    -- Format definition
    "name" varchar(100) NOT NULL,
    "description" text,
    "value_type" variable_value_type_enum NOT NULL,

    -- SQL expression for formatting
    "format_sql" text NOT NULL,

    -- Metadata
    "is_active" boolean DEFAULT true,
    "is_system" boolean DEFAULT false,

    -- Versioning
    "version" bigint DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,

    -- Constraints
    CONSTRAINT "pk_variable_formats" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uk_format_name" UNIQUE ("organization_id", "name"),
    CONSTRAINT "fk_variable_formats_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variable_formats_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for variable formats
CREATE INDEX "idx_variable_formats_org" ON "variable_formats"("organization_id", "value_type", "is_active");
CREATE INDEX "idx_variable_formats_name" ON "variable_formats"("name") WHERE "is_active" = true;

--bun:split
-- Create variables table
CREATE TABLE IF NOT EXISTS "variables"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,

    -- Variable definition
    "key" varchar(100) NOT NULL,
    "display_name" varchar(255) NOT NULL,
    "description" text,
    "category" varchar(100),

    -- SQL configuration
    "query" text NOT NULL,
    "applies_to" variable_context_enum NOT NULL,
    "required_params" jsonb DEFAULT '[]'::jsonb,

    -- Output configuration
    "default_value" text,
    "format_id" varchar(100),
    "value_type" variable_value_type_enum DEFAULT 'String',

    -- Metadata
    "is_active" boolean DEFAULT true,
    "is_system" boolean DEFAULT false,
    "is_validated" boolean DEFAULT false,
    "tags" text[],

    -- Versioning
    "version" bigint DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,

    -- Constraints
    CONSTRAINT "pk_variables" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uk_variable_key" UNIQUE ("organization_id", "key"),
    CONSTRAINT "fk_variables_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variables_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variables_format" FOREIGN KEY ("format_id", "business_unit_id", "organization_id")
        REFERENCES "variable_formats"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_variables_query_length" CHECK (length("query") > 0),
    CONSTRAINT "chk_variables_key_length" CHECK (length("key") >= 2 AND length("key") <= 100)
);

--bun:split
-- Indexes for fast lookups
CREATE INDEX "idx_variables_context" ON "variables"("organization_id", "applies_to", "is_active");
CREATE INDEX "idx_variables_key" ON "variables"("organization_id", "key") WHERE "is_active" = true;
CREATE INDEX "idx_variables_business_unit" ON "variables"("business_unit_id");
CREATE INDEX "idx_variables_created_updated" ON "variables"("created_at", "updated_at");

--bun:split
-- Index for tag searching
CREATE INDEX "idx_variables_tags" ON "variables" USING gin("tags") WHERE "tags" IS NOT NULL;

--bun:split
-- Table comments
COMMENT ON TABLE "variables" IS 'Stores variable definitions for template processing';

--bun:split
-- Column comments
COMMENT ON COLUMN "variables"."id" IS 'Unique identifier (pulid with var_ prefix)';
COMMENT ON COLUMN "variables"."key" IS 'Variable key used in templates (e.g., customerName)';
COMMENT ON COLUMN "variables"."display_name" IS 'User-friendly display name';
COMMENT ON COLUMN "variables"."query" IS 'Parameterized SQL query to fetch the variable value';
COMMENT ON COLUMN "variables"."applies_to" IS 'Context where this variable can be used';
COMMENT ON COLUMN "variables"."required_params" IS 'List of required parameters for the query';
COMMENT ON COLUMN "variables"."is_validated" IS 'Whether the query has been validated as safe';
COMMENT ON COLUMN "variables"."is_system" IS 'System-defined variables cannot be modified by users';

--bun:split
-- Trigger function for updating timestamps
CREATE OR REPLACE FUNCTION variables_update_timestamps()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS variables_update_trigger ON variables;

--bun:split
CREATE TRIGGER variables_update_trigger
    BEFORE UPDATE ON variables
    FOR EACH ROW
    EXECUTE FUNCTION variables_update_timestamps();

--bun:split
-- Performance optimization
ALTER TABLE "variables" ALTER COLUMN "organization_id" SET STATISTICS 1000;
ALTER TABLE "variables" ALTER COLUMN "applies_to" SET STATISTICS 500;
ALTER TABLE "variables" ALTER COLUMN "key" SET STATISTICS 500;
