-- Enums with documentation
CREATE TYPE "shipment_status_enum" AS ENUM(
    'New',
    -- Initial state when shipment is created
    'PartiallyAssigned',
    -- Shipment has been partially assigned to a worker
    'Assigned',
    -- Shipment has been assigned to a worker
    'InTransit',
    -- Shipment is currently being transported
    'Delayed',
    -- Shipment is currently delayed
    'PartiallyCompleted',
    -- Shipment has been partially completed
    'Completed',
    -- Shipment has been delivered successfully
    'ReadyToBill',
    -- Shipment is ready to be billed
    'ReviewRequired',
    -- Shipment requires review before being billed
    'Billed',
    -- Shipment has been billed to the customer
    'Canceled' -- Shipment has been Canceled
);

CREATE TYPE "rating_method_enum" AS ENUM(
    'FlatRate',
    -- Fixed rate for entire shipment
    'PerMile',
    -- Rate calculated per mile traveled
    'PerStop',
    -- Rate calculated per stop made
    'PerPound',
    -- Rate calculated by weight
    'PerPallet',
    -- Rate calculated by pallet position
    'PerLiearFoot',
    -- Rate calculated by linear feet of trailer space
    'Other', -- Custom rating method
    'FormulaTemplate' -- Formula template rating method
);

--bun:split
CREATE TABLE IF NOT EXISTS "shipments"(
    "id" varchar(100) NOT NULL,
    "pro_number" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" shipment_status_enum NOT NULL DEFAULT 'New',
    "bol" varchar(100) NOT NULL,
    "actual_ship_date" bigint,
    "actual_delivery_date" bigint,
    "temperature_min" temperature_fahrenheit,
    "temperature_max" temperature_fahrenheit,
    -- Billing Related Fields
    "rating_unit" integer NOT NULL DEFAULT 1 CHECK ("rating_unit" > 0),
    "rating_method" rating_method_enum NOT NULL DEFAULT 'FlatRate',
    "freight_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("freight_charge_amount" >= 0),
    "other_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("other_charge_amount" >= 0),
    "total_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("total_charge_amount" >= 0),
    "pieces" integer CHECK ("pieces" > 0),
    "weight" integer CHECK ("weight" > 0),
    -- Cancellation Related Fields
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "cancel_reason" varchar(100),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_bol" ON "shipments"("bol");

CREATE INDEX IF NOT EXISTS "idx_shipments_created_at" ON "shipments"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_shipments_status" ON "shipments"("status");

CREATE INDEX IF NOT EXISTS "idx_shipments_business_unit" ON "shipments"("business_unit_id", "organization_id");

COMMENT ON TABLE shipments IS 'Stores information about shipments and their billing status';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_search ON shipments USING GIN(search_vector);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_dates_brin ON shipments USING BRIN(actual_ship_date, actual_delivery_date, created_at) WITH (pages_per_range = 128);

--bun:split
CREATE OR REPLACE FUNCTION shipments_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.pro_number, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.bol, '')), 'A') || setweight(to_tsvector('english', COALESCE(CAST(NEW.status AS text), '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(NEW.rating_method AS text), '')), 'C');
    -- Update total_charge_amount if it's changed
    NEW.total_charge_amount := NEW.freight_charge_amount + NEW.other_charge_amount;
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS shipments_search_vector_trigger ON shipments;

--bun:split
CREATE TRIGGER shipments_search_vector_trigger
    BEFORE INSERT OR UPDATE ON shipments
    FOR EACH ROW
    EXECUTE FUNCTION shipments_search_vector_update();

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_active ON shipments(created_at DESC)
WHERE
    status NOT IN ('Completed', 'Canceled', 'Billed');

--bun:split
ALTER TABLE shipments
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE shipments
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE shipments
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_trgm_pro_bol ON shipments USING gin((pro_number || ' ' || bol) gin_trgm_ops);

--bun:split
CREATE INDEX idx_shipments_bu_org_status_created_at ON shipments(business_unit_id, organization_id, status, created_at DESC);

CREATE INDEX idx_shipments_bu_org_include ON shipments(business_unit_id, organization_id) INCLUDE (status, created_at, pro_number, bol);

--bun:split
ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS "owner_id" varchar(100);

-- Add relationship to users
ALTER TABLE shipments
    ADD CONSTRAINT "fk_shipments_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

