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

--bun:split
CREATE TABLE IF NOT EXISTS "stops"(
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
    "address_line" varchar(200),
    -- Metadata with generated columns for timestamp conversion
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "created_at_timestamp" timestamp GENERATED ALWAYS AS (to_timestamp(created_at)) STORED,
    "updated_at_timestamp" timestamp GENERATED ALWAYS AS (to_timestamp(updated_at)) STORED,
    CONSTRAINT "pk_stops" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    -- Added unique constraint for sequence within same shipment_move
    -- CONSTRAINT "uq_stops_shipment_move_sequence" UNIQUE ("shipment_move_id", "organization_id", "business_unit_id", "sequence"),
    CONSTRAINT "fk_stops_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Added check constraints for data validation
    CONSTRAINT "chk_stops_planned_times" CHECK (planned_departure >= planned_arrival),
    CONSTRAINT "chk_stops_actual_times" CHECK (actual_departure >= actual_arrival),
    CONSTRAINT "chk_stops_pieces_positive" CHECK (pieces IS NULL OR pieces >= 0),
    CONSTRAINT "chk_stops_weight_positive" CHECK (weight IS NULL OR weight >= 0)
);

-- Partial index for active stops (not canceled) to improve common queries
CREATE INDEX IF NOT EXISTS "idx_stops_active" ON "stops"("organization_id", "business_unit_id", "status")
WHERE
    status != 'Canceled';

-- Composite index for common filtering and sorting patterns
CREATE INDEX IF NOT EXISTS "idx_stops_common_queries" ON "stops"("organization_id", "business_unit_id", "created_at_timestamp", "status", "type");

-- Index for timestamp range queries
CREATE INDEX IF NOT EXISTS "idx_stops_timestamps" ON "stops"("created_at_timestamp", "updated_at_timestamp");

-- Index for shipment move relationship with included columns for common queries
CREATE INDEX IF NOT EXISTS "idx_stops_shipment_move" ON "stops"("shipment_move_id", "organization_id", "business_unit_id") INCLUDE ("status", "type", "sequence", "planned_arrival");

-- BRIN index for timestamp ranges (more efficient for large tables)
CREATE INDEX IF NOT EXISTS "idx_stops_brin_timestamps" ON "stops" USING BRIN("created_at_timestamp", "updated_at_timestamp") WITH (pages_per_range = 128);

COMMENT ON TABLE stops IS 'Stores information about pickup and delivery stops for shipments';

COMMENT ON COLUMN stops.created_at_timestamp IS 'Converted timestamp from created_at epoch for easier querying';

COMMENT ON COLUMN stops.updated_at_timestamp IS 'Converted timestamp from updated_at epoch for easier querying';

