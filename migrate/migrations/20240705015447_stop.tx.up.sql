DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'stop_type_enum') THEN
            CREATE TYPE stop_type_enum AS ENUM ('Pickup', 'SplitPickup', 'SplitDrop', 'Delivery', 'DropOff');
        END IF;
    END
$$;

-- bun:split

CREATE TABLE
    IF NOT EXISTS "stops"
(
    "id"                uuid                      NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"  uuid                      NOT NULL,
    "organization_id"   uuid                      NOT NULL,
    "status"            shipment_move_status_enum NOT NULL DEFAULT 'New',
    "sequence_number"   integer                   NOT NULL DEFAULT 1,
    "shipment_move_id"  uuid                      NOT NULL,
    "location_id"       uuid                      NOT NULL,
    "type"              stop_type_enum            NOT NULL,
    "pieces"            numeric(10, 2),
    "weight"            numeric(10, 2),
    "address_line"      TEXT,
    "planned_arrival"   TIMESTAMPTZ,
    "actual_arrival"    TIMESTAMPTZ,
    "planned_departure" TIMESTAMPTZ,
    "actual_departure"  TIMESTAMPTZ,
    "notes"             TEXT,
    "created_at"        TIMESTAMPTZ               NOT NULL DEFAULT current_timestamp,
    "updated_at"        TIMESTAMPTZ               NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("shipment_move_id") REFERENCES shipment_moves ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("location_id") REFERENCES locations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE INDEX idx_stop_shipment_move_id ON stops (shipment_move_id);
CREATE INDEX idx_stop_location_id ON stops (location_id);
CREATE INDEX idx_stop_org_bu ON stops (organization_id, business_unit_id);
CREATE INDEX idx_stop_created_at ON stops (created_at);

-- bun:split

COMMENT ON TABLE stops IS 'A record of a stop in a shipment move';
COMMENT ON COLUMN stops.id IS 'Unique identifier for the stop, generated as a UUID';
COMMENT ON COLUMN stops.business_unit_id IS 'Foreign key referencing the business unit that this stop belongs to';
COMMENT ON COLUMN stops.organization_id IS 'Foreign key referencing the organization that this stop belongs to';
COMMENT ON COLUMN stops.status IS 'The current status of the stop, using the shipment_move_status_enum (e.g., Pickup, SplitPickup, SplitDrop, Delivery, DropOff)';
COMMENT ON COLUMN stops.sequence_number IS 'The order of the stop in the shipment move';
COMMENT ON COLUMN stops.shipment_move_id IS 'Foreign key referencing the shipment move that this stop belongs to';
COMMENT ON COLUMN stops.location_id IS 'Foreign key referencing the location of the stop';
COMMENT ON COLUMN stops.type IS 'The type of stop, using the stop_type_enum (e.g., Pickup, SplitPickup, SplitDrop, Delivery, DropOff)';
COMMENT ON COLUMN stops.pieces IS 'The number of pieces at the stop';
COMMENT ON COLUMN stops.weight IS 'The weight of the stop';
COMMENT ON COLUMN stops.address_line IS 'The address of the stop';
COMMENT ON COLUMN stops.planned_arrival IS 'The planned arrival time at the stop';
COMMENT ON COLUMN stops.actual_arrival IS 'The actual arrival time at the stop';
COMMENT ON COLUMN stops.planned_departure IS 'The planned departure time from the stop';
COMMENT ON COLUMN stops.actual_departure IS 'The actual departure time from the stop';
COMMENT ON COLUMN stops.notes IS 'Notes about the stop';
COMMENT ON COLUMN stops.created_at IS 'Timestamp of when the stop was created, defaults to the current timestamp';
COMMENT ON COLUMN stops.updated_at IS 'Timestamp of the last update to the stop, defaults to the current timestamp';