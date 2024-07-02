CREATE TABLE
    IF NOT EXISTS "charge_types"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "name"             VARCHAR(50) NOT NULL,
    "description"      TEXT,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "charge_type_name_organization_id_unq" ON "charge_types" (LOWER("name"), organization_id);
CREATE INDEX idx_charge_types_name ON charge_types (name);
CREATE INDEX idx_charge_types_org_bu ON charge_types (organization_id, business_unit_id);
CREATE INDEX idx_charge_types_description ON charge_types USING GIN (description gin_trgm_ops);
CREATE INDEX idx_charge_types_created_at ON charge_types(created_at);

--bun:split

COMMENT ON COLUMN charge_types.id IS 'Unique identifier for the charge type, generated as a UUID';
COMMENT ON COLUMN charge_types.business_unit_id IS 'Foreign key referencing the business unit that this charge type belongs to';
COMMENT ON COLUMN charge_types.organization_id IS 'Foreign key referencing the organization that this charge type belongs to';
COMMENT ON COLUMN charge_types.status IS 'The current status of the charge type, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN charge_types.name IS 'A short, unique name for identifying the charge type, limited to 50 characters';
COMMENT ON COLUMN charge_types.description IS 'A detailed description of the charge type';
COMMENT ON COLUMN charge_types.created_at IS 'Timestamp of when the charge type was created, defaults to the current timestamp';
COMMENT ON COLUMN charge_types.updated_at IS 'Timestamp of the last update to the charge type, defaults to the current timestamp';