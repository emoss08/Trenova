CREATE TABLE
    IF NOT EXISTS "shipment_types"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10) NOT NULL,
    "color"            VARCHAR(10),
    "description"      TEXT,
    "version"          BIGINT      NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "shipment_types_code_organization_id_unq" ON "shipment_types" (LOWER("code"), organization_id);
CREATE INDEX idx_shipment_types_code ON shipment_types (code);
CREATE INDEX idx_shipment_types_org_bu ON shipment_types (organization_id, business_unit_id);
CREATE INDEX idx_shipment_types_description ON shipment_types USING GIN (description gin_trgm_ops);
CREATE INDEX idx_shipment_types_created_at ON shipment_types (created_at);

--bun:split

COMMENT ON COLUMN shipment_types.id IS 'Unique identifier for the shipment type, generated as a UUID';
COMMENT ON COLUMN shipment_types.business_unit_id IS 'Foreign key referencing the business unit that this shipment type belongs to';
COMMENT ON COLUMN shipment_types.organization_id IS 'Foreign key referencing the organization that this shipment type belongs to';
COMMENT ON COLUMN shipment_types.status IS 'The current status of the shipment type, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN shipment_types.code IS 'A short, unique code for identifying the shipment type, limited to 10 characters';
COMMENT ON COLUMN shipment_types.color IS 'A color code for the shipment type';
COMMENT ON COLUMN shipment_types.description IS 'A detailed description of the shipment type';
COMMENT ON COLUMN shipment_types.created_at IS 'Timestamp of when the shipment type was created, defaults to the current timestamp';
COMMENT ON COLUMN shipment_types.updated_at IS 'Timestamp of the last update to the shipment type, defaults to the current timestamp';