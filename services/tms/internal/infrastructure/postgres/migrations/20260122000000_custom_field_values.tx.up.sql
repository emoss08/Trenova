CREATE TABLE IF NOT EXISTS "custom_field_values"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "definition_id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "resource_id" varchar(100) NOT NULL,
    "value" jsonb NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,

    CONSTRAINT "pk_custom_field_values" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_cfv_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cfv_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cfv_definition" FOREIGN KEY ("definition_id", "organization_id", "business_unit_id")
        REFERENCES "custom_field_definitions"("id", "organization_id", "business_unit_id")
        ON DELETE CASCADE,
    CONSTRAINT "uq_cfv_resource_definition" UNIQUE (
        "organization_id", "business_unit_id", "resource_type", "resource_id", "definition_id"
    ),
    CONSTRAINT "chk_cfv_value_not_null" CHECK (value IS NOT NULL)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_cfv_resource" ON "custom_field_values"("resource_type", "resource_id");

CREATE INDEX IF NOT EXISTS "idx_cfv_definition" ON "custom_field_values"("definition_id");

CREATE INDEX IF NOT EXISTS "idx_cfv_tenant" ON "custom_field_values"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_cfv_resource_tenant" ON "custom_field_values"(
    "organization_id", "business_unit_id", "resource_type", "resource_id"
);

CREATE INDEX IF NOT EXISTS "idx_cfv_value" ON "custom_field_values" USING GIN("value");

COMMENT ON TABLE custom_field_values IS 'Centralized storage for custom field values across all entity types';

--bun:split
CREATE OR REPLACE FUNCTION custom_field_values_update_timestamp()
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
DROP TRIGGER IF EXISTS custom_field_values_update_timestamp_trigger ON custom_field_values;

CREATE TRIGGER custom_field_values_update_timestamp_trigger
    BEFORE UPDATE ON custom_field_values
    FOR EACH ROW
    EXECUTE FUNCTION custom_field_values_update_timestamp();

--bun:split
ALTER TABLE custom_field_values
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE custom_field_values
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE custom_field_values
    ALTER COLUMN resource_type SET STATISTICS 1000;

ALTER TABLE custom_field_values
    ALTER COLUMN resource_id SET STATISTICS 1000;
