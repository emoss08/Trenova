CREATE TABLE
    IF NOT EXISTS "reason_codes"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10) NOT NULL,
    "code_type"        VARCHAR(10),
    "description"      TEXT,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "reason_codes_code_organization_id_unq" ON "reason_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_reason_codes_code ON reason_codes (code);
CREATE INDEX idx_reason_codes_org_bu ON reason_codes (organization_id, business_unit_id);
CREATE INDEX idx_reason_codes_description ON reason_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_reason_codes_created_at ON reason_codes(created_at);

--bun:split

COMMENT ON COLUMN reason_codes.id IS 'Unique identifier for the reason code, generated as a UUID';
COMMENT ON COLUMN reason_codes.business_unit_id IS 'Foreign key referencing the business unit that this reason code belongs to';
COMMENT ON COLUMN reason_codes.organization_id IS 'Foreign key referencing the organization that this reason code belongs to';
COMMENT ON COLUMN reason_codes.status IS 'The current status of the reason code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN reason_codes.code IS 'A short, unique code for identifying the reason code, limited to 10 characters';
COMMENT ON COLUMN reason_codes.code_type IS 'The type of code, if applicable';
COMMENT ON COLUMN reason_codes.description IS 'A detailed description of the reason code';
COMMENT ON COLUMN reason_codes.created_at IS 'Timestamp of when the reason code was created, defaults to the current timestamp';
COMMENT ON COLUMN reason_codes.updated_at IS 'Timestamp of the last update to the reason code, defaults to the current timestamp';
