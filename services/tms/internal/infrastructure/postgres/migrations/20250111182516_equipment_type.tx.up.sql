--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE equipment_class_enum AS ENUM(
    'Tractor',
    'Trailer',
    'Container',
    'Other'
);

--bun:split
CREATE TABLE IF NOT EXISTS "equipment_types"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text,
    "class" equipment_class_enum NOT NULL,
    "color" varchar(10),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_equipment_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_equipment_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_equipment_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
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
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("class"), '')), 'C')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_search ON equipment_types USING GIN(search_vector);

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_code ON equipment_types USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_description ON equipment_types USING gin(description gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_equipment_types_trgm_code_description ON equipment_types USING gin((code || ' ' || description) gin_trgm_ops);
