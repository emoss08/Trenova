DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fuel_method_enum') THEN
            CREATE TYPE fuel_method_enum AS ENUM ('Distance', 'Flat', 'Percentage');
        END IF;
    END
$$;

--bun:split

CREATE TABLE
    IF NOT EXISTS "accessorial_charges"
(
    "id"               uuid             NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid             NOT NULL,
    "organization_id"  uuid             NOT NULL,
    "status"           status_enum      NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10)      NOT NULL,
    "description"      TEXT,
    "is_detention"     BOOLEAN          NOT NULL DEFAULT FALSE,
    "method"           fuel_method_enum NOT NULL,
    "shipment_id"      uuid             NOT NULL,
    "amount"           NUMERIC(19, 2)   NOT NULL DEFAULT 0,
    "created_at"       TIMESTAMPTZ      NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ      NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("shipment_id") REFERENCES shipments ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "accessorial_charges_code_organization_id_unq" ON "accessorial_charges" (LOWER("code"), organization_id);

--bun:split

COMMENT ON COLUMN accessorial_charges.id IS 'Unique identifier for the accessorial charge, generated as a UUID';
COMMENT ON COLUMN accessorial_charges.business_unit_id IS 'Foreign key referencing the business unit that this accessorial charge belongs to';
COMMENT ON COLUMN accessorial_charges.organization_id IS 'Foreign key referencing the organization that this accessorial charge belongs to';
COMMENT ON COLUMN accessorial_charges.status IS 'The current status of the accessorial charge, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN accessorial_charges.code IS 'A short, unique code for identifying the accessorial charge, limited to 10 characters';
COMMENT ON COLUMN accessorial_charges.description IS 'A detailed description of the accessorial charge';
COMMENT ON COLUMN accessorial_charges.is_detention IS 'Indicates whether the accessorial charge is for detention, represented as a boolean value';
COMMENT ON COLUMN accessorial_charges.method IS 'The method used for the accessorial charge, using the rating_method_enum (e.g., Distance, Flat, Percentage)';
COMMENT ON COLUMN accessorial_charges.amount IS 'The amount for the accessorial charge, represented as a numeric value with 19 digits and 2 decimal places, defaulting to 0';
COMMENT ON COLUMN accessorial_charges.created_at IS 'Timestamp of when the accessorial charge was created, defaults to the current timestamp';
COMMENT ON COLUMN accessorial_charges.updated_at IS 'Timestamp of the last update to the accessorial charge, defaults to the current timestamp';
COMMENT ON TABLE accessorial_charges IS 'A table to store information about accessorial charges, which are additional fees or charges associated with a shipment';