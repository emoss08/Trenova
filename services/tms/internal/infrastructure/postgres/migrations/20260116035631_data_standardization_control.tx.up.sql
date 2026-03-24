CREATE TYPE "case_format_enum" AS ENUM(
    'AsEntered',
    'Upper',
    'Lower',
    'TitleCase'
);

--bun:split
CREATE TABLE IF NOT EXISTS "data_entry_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code_case" "case_format_enum" NOT NULL DEFAULT 'Upper',
    "name_case" "case_format_enum" NOT NULL DEFAULT 'TitleCase',
    "email_case" "case_format_enum" NOT NULL DEFAULT 'Lower',
    "city_case" "case_format_enum" NOT NULL DEFAULT 'TitleCase',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_data_entry_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_data_entry_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_data_entry_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_data_entry_controls_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_data_entry_controls_business_unit" ON "data_entry_controls"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_data_entry_controls_created_at" ON "data_entry_controls"("created_at", "updated_at");

COMMENT ON TABLE data_entry_controls IS 'Stores configuration for data entry standardization and transformation rules';

--bun:split
CREATE OR REPLACE FUNCTION data_entry_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS data_entry_controls_update_timestamp_trigger ON data_entry_controls;

CREATE TRIGGER data_entry_controls_update_timestamp_trigger
    BEFORE UPDATE ON data_entry_controls
    FOR EACH ROW
    EXECUTE FUNCTION data_entry_controls_update_timestamp();

--bun:split
ALTER TABLE data_entry_controls
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE data_entry_controls
    ALTER COLUMN business_unit_id SET STATISTICS 1000;
