CREATE TYPE "shipment_status_enum" AS ENUM(
    -- Initial state when shipment is created
    'New',
    -- Shipment has been partially assigned to a worker
    'PartiallyAssigned',
    -- Shipment has been assigned to a worker
    'Assigned',
    -- Shipment is currently being transported
    'InTransit',
    -- Shipment is currently delayed
    'Delayed',
    -- Shipment has been partially completed
    'PartiallyCompleted',
    -- Shipment is completed
    'Completed',
    -- Shipment is ready to be invoiced
    'ReadyToInvoice',
    -- Shipment has been invoiced to the customer
    'Invoiced',
    -- Shipment has been Canceled
    'Canceled'
);

--bun:split
CREATE TABLE IF NOT EXISTS "shipments"(
    "id" varchar(100) NOT NULL,
    "pro_number" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" shipment_status_enum NOT NULL DEFAULT 'New',
    "bol" varchar(100),
    "actual_ship_date" bigint,
    "actual_delivery_date" bigint,
    "temperature_min" temperature_fahrenheit,
    "temperature_max" temperature_fahrenheit,
    -- Billing Related Fields
    "rating_unit" integer NOT NULL DEFAULT 1 CHECK ("rating_unit" > 0),
    "freight_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("freight_charge_amount" >= 0),
    "other_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("other_charge_amount" >= 0),
    "total_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("total_charge_amount" >= 0),
    "pieces" integer CHECK ("pieces" > 0),
    "weight" integer CHECK ("weight" > 0),
    -- Cancellation Related Fields
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "cancel_reason" varchar(100),
    "entered_by_id" varchar(100),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_entered_by" FOREIGN KEY ("entered_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_shipments_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_bol" ON "shipments"("bol");

CREATE INDEX IF NOT EXISTS "idx_shipments_created_at" ON "shipments"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_shipments_status" ON "shipments"("status");

CREATE INDEX IF NOT EXISTS "idx_shipments_entered_by" ON "shipments"("entered_by_id");

CREATE INDEX IF NOT EXISTS "idx_shipments_business_unit" ON "shipments"("business_unit_id", "organization_id");

COMMENT ON TABLE shipments IS 'Stores information about shipments and their billing status';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("pro_number", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("bol", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("status"), '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_search ON shipments USING GIN(search_vector);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_dates_brin ON shipments USING BRIN(actual_ship_date, actual_delivery_date, created_at) WITH (pages_per_range = 128);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_active ON shipments(organization_id, created_at DESC)
WHERE
    status NOT IN ('Completed', 'Invoiced', 'Canceled');

CREATE INDEX IF NOT EXISTS idx_shipments_in_transit ON shipments(organization_id, business_unit_id)
WHERE
    status = 'InTransit';

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipments_trgm_pro_bol ON shipments USING gin((pro_number || ' ' || bol) gin_trgm_ops);

--bun:split
CREATE INDEX idx_shipments_bu_org_status_created_at ON shipments(business_unit_id, organization_id, status, created_at DESC);

CREATE INDEX idx_shipments_bu_org_include ON shipments(business_unit_id, organization_id) INCLUDE (status, created_at, pro_number, bol);

--bun:split
CREATE STATISTICS IF NOT EXISTS shipments_status_org_stats (dependencies)
    ON status, organization_id FROM shipments;

CREATE STATISTICS IF NOT EXISTS shipments_status_bu_stats (dependencies)
    ON status, business_unit_id FROM shipments;

--bun:split
ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS "owner_id" varchar(100);

-- Add relationship to users
ALTER TABLE shipments
    ADD CONSTRAINT "fk_shipments_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

