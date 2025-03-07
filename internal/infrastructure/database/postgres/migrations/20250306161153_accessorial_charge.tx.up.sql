-- Statement # 1
-- Enums with documentation
CREATE TYPE "accessorial_method_enum" AS ENUM (
    'Flat',
    'Distance',
    'Percentage'
);

-- Statement # 2
-- bun:split
CREATE TABLE IF NOT EXISTS "accessorial_charges" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text NOT NULL,
    "unit" integer NOT NULL,
    "method" accessorial_method_enum NOT NULL,
    "amount" numeric(19, 4) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_accessorial_charges" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_accessorial_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accessorial_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Statement # 3
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_status" ON "accessorial_charges" ("status");

-- Statement # 4
-- bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_business_unit" ON "accessorial_charges" ("business_unit_id", "organization_id");

-- Statement # 5
COMMENT ON TABLE accessorial_charges IS 'Stores information about accessorial charges';

-- Statement # 6
-- bun:split
ALTER TABLE "accessorial_charges"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Statement # 7
-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_search ON accessorial_charges USING GIN (search_vector);

-- Statement # 8
-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_dates_brin ON accessorial_charges USING BRIN (created_at, updated_at) WITH (pages_per_range = 128);

-- Statement # 9
-- bun:split
CREATE OR REPLACE FUNCTION accessorial_charges_search_vector_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', coalesce(NEW.code, '')), 'A') || setweight(to_tsvector('simple', coalesce(NEW.description, '')), 'B');
    -- Auto-update timestamps
    NEW.updated_at := extract(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

-- Statement # 10
--bun:split
DROP TRIGGER IF EXISTS accessorial_charges_search_vector_trigger ON accessorial_charges;

-- Statement # 11
--bun:split
CREATE TRIGGER accessorial_charges_search_vector_trigger
    BEFORE INSERT OR UPDATE ON accessorial_charges
    FOR EACH ROW
    EXECUTE FUNCTION accessorial_charges_search_vector_update ();

-- Statement # 12
-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_active ON accessorial_charges (created_at DESC)
WHERE
    status != 'Inactive';

-- Statement # 13
-- bun:split
ALTER TABLE accessorial_charges
    ALTER COLUMN status SET STATISTICS 1000;

-- Statement # 14
-- bun:split
ALTER TABLE accessorial_charges
    ALTER COLUMN organization_id SET STATISTICS 1000;

-- Statement # 15
-- bun:split
ALTER TABLE accessorial_charges
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

-- Statement # 16
-- bun:split
CREATE INDEX IF NOT EXISTS idx_accessorial_charges_trgm_code_description ON accessorial_charges USING gin ((code || ' ' || description) gin_trgm_ops);

