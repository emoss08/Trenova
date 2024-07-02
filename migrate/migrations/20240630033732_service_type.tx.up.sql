CREATE TABLE
    IF NOT EXISTS "service_types"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10) NOT NULL,
    "description"      TEXT,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "service_types_code_organization_id_unq" ON "service_types" (LOWER("code"), organization_id);
CREATE INDEX idx_service_types_code ON service_types (code);
CREATE INDEX idx_service_types_org_bu ON service_types (organization_id, business_unit_id);
CREATE INDEX idx_service_types_description ON service_types USING GIN (description gin_trgm_ops);
CREATE INDEX idx_service_types_created_at ON service_types(created_at);

--bun:split

COMMENT ON COLUMN service_types.id IS 'Unique identifier for the service type, generated as a UUID';
COMMENT ON COLUMN service_types.business_unit_id IS 'Foreign key referencing the business unit that this service type belongs to';
COMMENT ON COLUMN service_types.organization_id IS 'Foreign key referencing the organization that this service type belongs to';
COMMENT ON COLUMN service_types.status IS 'The current status of the service type, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN service_types.code IS 'A short, unique code for identifying the service type, limited to 10 characters';
COMMENT ON COLUMN service_types.description IS 'A detailed description of the service type';
COMMENT ON COLUMN service_types.created_at IS 'Timestamp of when the service type was created, defaults to the current timestamp';
COMMENT ON COLUMN service_types.updated_at IS 'Timestamp of the last update to the service type, defaults to the current timestamp';
