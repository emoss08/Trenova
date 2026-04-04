CREATE TYPE "accessorial_method_enum" AS ENUM(
    'Flat',
    'PerUnit',
    'Percentage'
);

CREATE TYPE "rate_unit_enum" AS ENUM(
    'Mile',
    'Hour',
    'Day',
    'Stop'
);

-- bun:split
CREATE TABLE IF NOT EXISTS "accessorial_charges"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text NOT NULL,
    "rate_unit" rate_unit_enum NULL,
    "method" accessorial_method_enum NOT NULL,
    "amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accessorial_charges" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_accessorial_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accessorial_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_status" ON "accessorial_charges"("status");

-- bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_business_unit" ON "accessorial_charges"("business_unit_id", "organization_id");

COMMENT ON TABLE accessorial_charges IS 'Stores information about accessorial charges';

-- bun:split
ALTER TABLE "accessorial_charges"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_search ON accessorial_charges USING GIN(search_vector);

-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_dates_brin ON accessorial_charges USING BRIN(created_at, updated_at) WITH (pages_per_range = 128);

-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_active ON accessorial_charges(created_at DESC)
WHERE
    status != 'Inactive';

-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_trgm_code_description ON accessorial_charges USING gin((code || ' ' || description) gin_trgm_ops);

--bun:split
ALTER TABLE "shipment_controls"
    ADD COLUMN IF NOT EXISTS detention_charge_id varchar(100) NULL;

--bun:split
ALTER TABLE "shipment_controls"
    ADD CONSTRAINT "fk_shipment_controls_detention_charge" FOREIGN KEY ("detention_charge_id", "organization_id", "business_unit_id") REFERENCES "accessorial_charges"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

