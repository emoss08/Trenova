CREATE TABLE
    IF NOT EXISTS "equipment_manufacturers"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "name"             VARCHAR     NOT NULL,
    "description"      TEXT,
    "version"          BIGINT      NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "equipment_manufacturers_name_organization_id_unq" ON "equipment_manufacturers" (LOWER("name"), organization_id);
CREATE INDEX idx_equipment_manufacturers_name ON equipment_manufacturers (name);
CREATE INDEX idx_equipment_manufacturers_org_bu ON equipment_manufacturers (organization_id, business_unit_id);
CREATE INDEX idx_equipment_manufacturers_description ON equipment_manufacturers USING GIN (description gin_trgm_ops);
CREATE INDEX idx_equipment_manufacturers_created_at ON equipment_manufacturers (created_at);

--bun:split

COMMENT ON COLUMN equipment_manufacturers.id IS 'Unique identifier for the equipment manufacturer, generated as a UUID';
COMMENT ON COLUMN equipment_manufacturers.business_unit_id IS 'Foreign key referencing the business unit that this equipment manufacturer belongs to';
COMMENT ON COLUMN equipment_manufacturers.organization_id IS 'Foreign key referencing the organization that this equipment manufacturer belongs to';
COMMENT ON COLUMN equipment_manufacturers.status IS 'The current status of the equipment manufacturer, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN equipment_manufacturers.name IS 'A short, unique name for identifying the equipment manufacturer';
COMMENT ON COLUMN equipment_manufacturers.description IS 'A detailed description of the equipment manufacturer';
COMMENT ON COLUMN equipment_manufacturers.created_at IS 'Timestamp of when the equipment manufacturer was created, defaults to the current timestamp';
COMMENT ON COLUMN equipment_manufacturers.updated_at IS 'Timestamp of the last update to the equipment manufacturer, defaults to the current timestamp';
