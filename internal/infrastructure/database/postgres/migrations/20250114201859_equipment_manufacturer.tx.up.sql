-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

CREATE TABLE IF NOT EXISTS "equipment_manufacturers" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(100) NOT NULL,
    "description" text,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_equipment_manu" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_equipment_manu_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_equipment_manu_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for equipment_manufacturers table
CREATE UNIQUE INDEX "idx_equipment_manu_name" ON "equipment_manufacturers" (lower("name"), "organization_id");

CREATE INDEX "idx_equipment_manu_business_unit" ON "equipment_manufacturers" ("business_unit_id");

CREATE INDEX "idx_equipment_manu_organization" ON "equipment_manufacturers" ("organization_id");

CREATE INDEX "idx_equipment_manu_created_updated" ON "equipment_manufacturers" ("created_at", "updated_at");

COMMENT ON TABLE "equipment_manufacturers" IS 'Stores information about equipment manufacturers';

