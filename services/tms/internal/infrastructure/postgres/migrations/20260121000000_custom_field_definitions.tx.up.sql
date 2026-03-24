CREATE TYPE "custom_field_type_enum" AS ENUM(
    'text',
    'number',
    'date',
    'boolean',
    'select',
    'multiSelect'
);

--bun:split
CREATE TABLE IF NOT EXISTS "custom_field_definitions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "label" varchar(150) NOT NULL,
    "description" text,
    "field_type" custom_field_type_enum NOT NULL,
    "is_required" boolean NOT NULL DEFAULT FALSE,
    "is_active" boolean NOT NULL DEFAULT TRUE,
    "display_order" integer NOT NULL DEFAULT 0,
    "color" varchar(20),
    "options" jsonb DEFAULT '[]'::jsonb,
    "validation_rules" jsonb DEFAULT '{}'::jsonb,
    "default_value" jsonb DEFAULT NULL,
    "ui_attributes" jsonb DEFAULT '{}'::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_custom_field_definitions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_custom_field_definitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_custom_field_definitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_custom_field_definitions_name" UNIQUE ("organization_id", "business_unit_id", "resource_type", "name"),
    CONSTRAINT "chk_custom_field_name_format" CHECK (name ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT "chk_custom_field_options_format" CHECK (jsonb_typeof(options) = 'array'),
    CONSTRAINT "chk_custom_field_validation_format" CHECK (jsonb_typeof(validation_rules) = 'object')
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_cfd_resource_type" ON "custom_field_definitions"("resource_type");

CREATE INDEX IF NOT EXISTS "idx_cfd_org_bu" ON "custom_field_definitions"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_cfd_active" ON "custom_field_definitions"("is_active") WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS "idx_cfd_resource_active" ON "custom_field_definitions"("organization_id", "business_unit_id", "resource_type") WHERE is_active = TRUE;

COMMENT ON TABLE custom_field_definitions IS 'Stores custom field definitions for various resource types';

--bun:split
CREATE OR REPLACE FUNCTION custom_field_definitions_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    NEW.version := OLD.version + 1;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS custom_field_definitions_update_timestamp_trigger ON custom_field_definitions;

CREATE TRIGGER custom_field_definitions_update_timestamp_trigger
    BEFORE UPDATE ON custom_field_definitions
    FOR EACH ROW
    EXECUTE FUNCTION custom_field_definitions_update_timestamp();

--bun:split
ALTER TABLE custom_field_definitions
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE custom_field_definitions
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE custom_field_definitions
    ALTER COLUMN resource_type SET STATISTICS 1000;
