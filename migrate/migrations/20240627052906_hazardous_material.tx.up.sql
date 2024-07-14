CREATE TABLE
    IF NOT EXISTS "hazardous_materials"
(
    "id"                   uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"     uuid        NOT NULL,
    "organization_id"      uuid        NOT NULL,
    "name"                 VARCHAR(50) NOT NULL,
    "status"               status_enum NOT NULL DEFAULT 'Active',
    "hazard_class"         VARCHAR(16) NOT NULL DEFAULT 'HazardClass1And1',
    "erg_number"           VARCHAR,
    "description"          TEXT,
    "packing_group"        VARCHAR,
    "proper_shipping_name" TEXT,
    "version"              BIGINT      NOT NULL,
    "created_at"           TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"           TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "hazardous_materials_name_organization_id_unq" ON "hazardous_materials" (LOWER("name"), organization_id);
CREATE INDEX idx_hazardous_materials_name ON hazardous_materials (name);
CREATE INDEX idx_hazardous_materials_org_bu ON hazardous_materials (organization_id, business_unit_id);
CREATE INDEX idx_hazardous_materials_description ON hazardous_materials USING GIN (description gin_trgm_ops);
CREATE INDEX idx_hazardous_materials_created_at ON hazardous_materials (created_at);

--bun:split

COMMENT ON COLUMN hazardous_materials.id IS 'Unique identifier for the hazardous material, generated as a UUID';
COMMENT ON COLUMN hazardous_materials.business_unit_id IS 'Foreign key referencing the business unit that this hazardous material belongs to';
COMMENT ON COLUMN hazardous_materials.organization_id IS 'Foreign key referencing the organization that this hazardous material belongs to';
COMMENT ON COLUMN hazardous_materials.status IS 'The current status of the hazardous material, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN hazardous_materials.name IS 'A short, unique name for identifying the hazardous material, limited to 50 characters';
COMMENT ON COLUMN hazardous_materials.hazard_class IS 'The hazard class of the hazardous material, using the Hazard Class number (e.g., 1.1)';
COMMENT ON COLUMN hazardous_materials.erg_number IS 'The Emergency Response Guidebook (ERG) number for the hazardous material';
COMMENT ON COLUMN hazardous_materials.description IS 'A detailed description of the hazardous material';
COMMENT ON COLUMN hazardous_materials.packing_group IS 'The packing group of the hazardous material';
COMMENT ON COLUMN hazardous_materials.proper_shipping_name IS 'The proper shipping name of the hazardous material';
COMMENT ON COLUMN hazardous_materials.created_at IS 'Timestamp of when the hazardous material was created, defaults to the current timestamp';
COMMENT ON COLUMN hazardous_materials.updated_at IS 'Timestamp of the last update to the hazardous material, defaults to the current timestamp';
