CREATE TYPE hazardous_class_enum AS ENUM(
    'HazardClass1And1',
    'HazardClass1And2',
    'HazardClass1And3',
    'HazardClass1And4',
    'HazardClass1And5',
    'HazardClass1And6',
    'HazardClass2And1',
    'HazardClass2And2',
    'HazardClass2And3',
    'HazardClass3',
    'HazardClass4And1',
    'HazardClass4And2',
    'HazardClass4And3',
    'HazardClass5And1',
    'HazardClass5And2',
    'HazardClass6And1',
    'HazardClass6And2',
    'HazardClass7',
    'HazardClass8'
);

--bun:split

CREATE TYPE packing_group_enum AS ENUM(
    'I',
    'II',
    'III'
);

--bun:split

CREATE TABLE IF NOT EXISTS "hazardous_materials"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "class" hazardous_class_enum NOT NULL,
    "un_number" varchar(100),
    "erg_number" varchar(100),
    "packing_group" packing_group_enum,
    "proper_shipping_name" text NOT NULL,
    "handling_instructions" text NOT NULL,
    "emergency_contact" text NOT NULL,
    "emergency_contact_phone_number" text NOT NULL,
    "placard_required" boolean NOT NULL DEFAULT FALSE,
    "is_reportable_quantity" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_hazardous_materials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_hazardous_materials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hazardous_materials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for hazardous_materials table
CREATE UNIQUE INDEX "idx_hazardous_materials_code" ON "hazardous_materials"(lower("code"), "organization_id");

CREATE INDEX "idx_hazardous_materials_business_unit" ON "hazardous_materials"("business_unit_id");

CREATE INDEX "idx_hazardous_materials_organization" ON "hazardous_materials"("organization_id");

CREATE INDEX "idx_hazardous_materials_created_updated" ON "hazardous_materials"("created_at", "updated_at");

COMMENT ON TABLE "hazardous_materials" IS 'Stores information about hazardous materials';

