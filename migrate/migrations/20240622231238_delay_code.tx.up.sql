CREATE TABLE
    IF NOT EXISTS "delay_codes"
(
    "created_at"          TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"          TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"                  uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "status"              status_enum NOT NULL DEFAULT 'Active',
    "code"                VARCHAR(10) NOT NULL,
    "f_carrier_or_driver" BOOLEAN     NOT NULL DEFAULT FALSE,
    "description"         TEXT        NOT NULL,
    "color"               VARCHAR(10),
    "business_unit_id"    uuid        NOT NULL,
    "organization_id"     uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "delay_code_code_organization_id_unq" ON "delay_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_delay_codes_code ON delay_codes (code);
CREATE INDEX idx_delay_codes_org_bu ON delay_codes (organization_id, business_unit_id);
CREATE INDEX idx_delay_codes_description ON delay_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_delay_codes_created_at ON delay_codes(created_at);

--bun:split

COMMENT ON COLUMN delay_codes.id IS 'Unique identifier for the delay code, generated as a UUID';
COMMENT ON COLUMN delay_codes.business_unit_id IS 'Foreign key referencing the business unit that this delay code belongs to';
COMMENT ON COLUMN delay_codes.organization_id IS 'Foreign key referencing the organization that this delay code belongs to';
COMMENT ON COLUMN delay_codes.status IS 'The current status of the delay code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN delay_codes.code IS 'A short, unique code for identifying the delay code, limited to 10 characters';
COMMENT ON COLUMN delay_codes.f_carrier_or_driver IS 'Indicates if the delay code is for the carrier or driver';
COMMENT ON COLUMN delay_codes.description IS 'A detailed description of the delay code';
COMMENT ON COLUMN delay_codes.color IS 'The color associated with the delay code';
COMMENT ON COLUMN delay_codes.created_at IS 'Timestamp of when the delay code was created, defaults to the current timestamp';
COMMENT ON COLUMN delay_codes.updated_at IS 'Timestamp of the last update to the delay code, defaults to the current timestamp';