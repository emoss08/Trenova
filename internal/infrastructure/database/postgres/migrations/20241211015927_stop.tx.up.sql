-- Enums with comments explaining each type
CREATE TYPE "stop_status_enum" AS ENUM(
    'New', -- Initial state when move is created
    'InTransit', -- Move is currently being executed
    'Completed', -- Move has been completed successfully
    'Canceled' -- Move has been cancelled and won't be completed
);

--bun:split
CREATE TYPE stop_type_enum AS ENUM(
    'Pickup', -- Regular pickup stop
    'Delivery', -- Regular delivery stop
    'SplitDelivery', -- Partial delivery of shipment
    'SplitPickup' -- Partial pickup of shipment
);

CREATE TABLE IF NOT EXISTS stops(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    -- Core fields
    "status" stop_status_enum NOT NULL DEFAULT 'New',
    "type" stop_type_enum NOT NULL DEFAULT 'Pickup',
    "sequence" integer NOT NULL DEFAULT 0,
    "pieces" integer,
    "weight" integer,
    "planned_arrival" bigint NOT NULL,
    "planned_departure" bigint NOT NULL,
    "actual_arrival" bigint,
    "actual_departure" bigint,
    "address_line" varchar(255),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_stops" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stops_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_stops_created_at" ON "stops"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_stops_status" ON "stops"("status");

CREATE INDEX IF NOT EXISTS "idx_stops_type" ON "stops"("type");

CREATE INDEX IF NOT EXISTS "idx_stops_business_unit" ON "stops"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_stops_shipment_move" ON "stops"("shipment_move_id", "organization_id", "business_unit_id");

COMMENT ON TABLE stops IS 'Stores information about pickup and delivery stops for shipments';

