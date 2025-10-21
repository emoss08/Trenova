--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Equipment class enum with descriptions
CREATE TYPE equipment_class_enum AS ENUM(
    'Tractor',
    'Trailer',
    'Container',
    'Other'
);

--bun:split
CREATE TABLE IF NOT EXISTS "equipment_types"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text,
    "class" equipment_class_enum NOT NULL,
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_equipment_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_equipment_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_equipment_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for equipment_types table
-- Ensure that the code is unique for each organization
CREATE UNIQUE INDEX "idx_equipment_types_code" ON "equipment_types"(lower("code"), "organization_id");

CREATE INDEX "idx_equipment_types_business_unit" ON "equipment_types"("business_unit_id");

CREATE INDEX "idx_equipment_types_organization" ON "equipment_types"("organization_id");

CREATE INDEX "idx_equipment_types_color" ON "equipment_types"("color");

CREATE INDEX "idx_equipment_types_created_updated" ON "equipment_types"("created_at", "updated_at");

COMMENT ON TABLE "equipment_types" IS 'Stores information about equipment types';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "tractor_type_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_tractor_type" FOREIGN KEY ("tractor_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "trailer_type_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_trailer_type" FOREIGN KEY ("trailer_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "equipment_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_search ON equipment_types USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION equipment_types_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(NEW.class AS text), '')), 'C');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS equipment_types_search_vector_trigger ON equipment_types;

--bun:split
CREATE TRIGGER equipment_types_search_vector_trigger
    BEFORE INSERT OR UPDATE ON equipment_types
    FOR EACH ROW
    EXECUTE FUNCTION equipment_types_search_vector_update();

--bun:split
ALTER TABLE equipment_types
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE equipment_types
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE equipment_types
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE equipment_types
    ALTER COLUMN class SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_code ON equipment_types USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_description ON equipment_types USING gin(description gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_code_description ON equipment_types USING gin((code || ' ' || description) gin_trgm_ops);

