-- Enums with documentation
CREATE TYPE "shipment_status_enum" AS ENUM(
    'New', -- Initial state when shipment is created
    'InTransit', -- Shipment is currently being transported
    'Completed', -- Shipment has been delivered successfully
    'Billed', -- Shipment has been billed to the customer
    'Canceled' -- Shipment has been Canceled
);

CREATE TYPE "rating_method_enum" AS ENUM(
    'Flat', -- Fixed rate for entire shipment
    'PerMile', -- Rate calculated per mile traveled
    'PerStop', -- Rate calculated per stop made
    'PerPound', -- Rate calculated by weight
    'PerPallet', -- Rate calculated by pallet position
    'PerLinearFoot', -- Rate calculated by linear feet of trailer space
    'Other' -- Custom rating method
);

--bun:split
CREATE TABLE IF NOT EXISTS "shipments"(
    "id" varchar(100) NOT NULL,
    "pro_number" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" shipment_status_enum NOT NULL DEFAULT 'New',
    "bol" varchar(100) NOT NULL,
    "rating_method" rating_method_enum NOT NULL DEFAULT 'Flat',
    "rating_unit" integer NOT NULL DEFAULT 1 CHECK ("rating_unit" > 0),
    "freight_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("freight_charge_amount" >= 0),
    "other_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("other_charge_amount" >= 0),
    "total_charge_amount" numeric(19, 4) NOT NULL DEFAULT 0 CHECK ("total_charge_amount" >= 0),
    "pieces" integer CHECK ("pieces" > 0),
    "weight" integer CHECK ("weight" > 0),
    "temperature_min" numeric(10, 2),
    "temperature_max" numeric(10, 2),
    "bill_date" bigint,
    "ready_to_bill" boolean NOT NULL DEFAULT FALSE,
    "ready_to_bill_date" bigint,
    "sent_to_billing" boolean NOT NULL DEFAULT FALSE,
    "sent_to_billing_date" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_bol" ON "shipments"("bol");

CREATE INDEX IF NOT EXISTS "idx_shipments_created_at" ON "shipments"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_shipments_status" ON "shipments"("status");

CREATE INDEX IF NOT EXISTS "idx_shipments_business_unit" ON "shipments"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_shipments_billing_status" ON "shipments"("ready_to_bill", "sent_to_billing");

-- Add helpful comments
COMMENT ON TABLE shipments IS 'Stores information about shipments and their billing status';

