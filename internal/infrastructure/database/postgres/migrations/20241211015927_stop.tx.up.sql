-- Enums with comments explaining each type
CREATE TYPE stop_type_enum AS ENUM(
    'Pickup', -- Regular pickup stop
    'Delivery', -- Regular delivery stop
    'SplitDrop', -- Partial delivery of shipment
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
    "pieces" integer CHECK ("pieces" > 0),
    "weight" integer CHECK ("weight" > 0),
    "scheduled_arrival_date" bigint NOT NULL,
    "scheduled_departure_date" bigint NOT NULL,
    "actual_arrival_date" bigint,
    "actual_departure_date" bigint,
    "address_line" varchar(100) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_stops" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stops_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_scheduled_dates" CHECK ("scheduled_departure_date" >= "scheduled_arrival_date"),
    CONSTRAINT "check_actual_dates" CHECK ("actual_departure_date" >= "actual_arrival_date")
);

CREATE INDEX IF NOT EXISTS "idx_stops_created_at" ON "stops"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_stops_status" ON "stops"("status");

CREATE INDEX IF NOT EXISTS "idx_stops_type" ON "stops"("type");

CREATE INDEX IF NOT EXISTS "idx_stops_business_unit" ON "stops"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_stops_shipment_move" ON "stops"("shipment_move_id", "organization_id");

COMMENT ON TABLE stops IS 'Stores information about pickup and delivery stops for shipments';

