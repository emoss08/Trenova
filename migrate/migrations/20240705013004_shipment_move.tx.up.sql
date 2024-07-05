DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'shipment_move_status_enum') THEN
            CREATE TYPE shipment_move_status_enum AS ENUM ('New', 'InProgress', 'Completed', 'Voided');
        END IF;
    END
$$;

-- bun:split

CREATE TABLE
    IF NOT EXISTS "shipment_moves"
(
    "id"                  uuid                      NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"    uuid                      NOT NULL,
    "organization_id"     uuid                      NOT NULL,
    "status"              shipment_move_status_enum NOT NULL DEFAULT 'New',
    "is_loaded"           boolean                   NOT NULL DEFAULT false,
    "sequence_number"     integer                   NOT NULL DEFAULT 1,
    "estimated_distance"  numeric(10, 2),
    "actual_distance"     numeric(10, 2),
    "estimated_cost"      numeric(19, 4),
    "actual_cost"         numeric(19, 4),
    "notes"               TEXT,
    "shipment_id"         uuid                      NOT NULL,
    "tractor_id"          uuid                      NOT NULL,
    "trailer_id"          uuid                      NOT NULL,
    "primary_worker_id"   uuid                      NOT NULL,
    "secondary_worker_id" uuid,
    "created_at"          TIMESTAMPTZ               NOT NULL DEFAULT current_timestamp,
    "updated_at"          TIMESTAMPTZ               NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("shipment_id") REFERENCES shipments ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("tractor_id") REFERENCES tractors ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("trailer_id") REFERENCES trailers ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("primary_worker_id") REFERENCES workers ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("secondary_worker_id") REFERENCES workers ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX idx_shipment_move_shipment_id ON shipment_moves (shipment_id);
CREATE INDEX idx_shipment_move_tractor_id ON shipment_moves (tractor_id);
CREATE INDEX idx_shipment_move_trailer_id ON shipment_moves (trailer_id);
CREATE INDEX idx_shipment_move_primary_worker_id ON shipment_moves (primary_worker_id);
CREATE INDEX idx_shipment_move_secondary_worker_id ON shipment_moves (secondary_worker_id);
CREATE INDEX idx_shipment_move_org_bu ON shipment_moves (organization_id, business_unit_id);
CREATE INDEX idx_shipment_move_created_at ON shipment_moves (created_at);

--bun:split

COMMENT ON TABLE shipment_moves IS 'A record of the movement of a shipment from origin to destination';
COMMENT ON COLUMN shipment_moves.id IS 'Unique identifier for the shipment move, generated as a UUID';
COMMENT ON COLUMN shipment_moves.business_unit_id IS 'Foreign key referencing the business unit that this shipment move belongs to';
COMMENT ON COLUMN shipment_moves.organization_id IS 'Foreign key referencing the organization that this shipment move belongs to';
COMMENT ON COLUMN shipment_moves.status IS 'The current status of the shipment move, using the shipment_move_status_enum (e.g., New, InProgress, Completed)';
COMMENT ON COLUMN shipment_moves.is_loaded IS 'A flag indicating whether the shipment has been loaded onto the trailer';
COMMENT ON COLUMN shipment_moves.sequence_number IS 'The sequence number for the shipment move';
COMMENT ON COLUMN shipment_moves.estimated_distance IS 'The estimated distance for the shipment move';
COMMENT ON COLUMN shipment_moves.actual_distance IS 'The actual distance for the shipment move';
COMMENT ON COLUMN shipment_moves.estimated_cost IS 'The estimated cost for the shipment move';
COMMENT ON COLUMN shipment_moves.actual_cost IS 'The actual cost for the shipment move';
COMMENT ON COLUMN shipment_moves.notes IS 'Any notes for the shipment move';
COMMENT ON COLUMN shipment_moves.shipment_id IS 'Foreign key referencing the shipment that this move is associated with';
COMMENT ON COLUMN shipment_moves.tractor_id IS 'Foreign key referencing the tractor that is being used for the shipment move';
COMMENT ON COLUMN shipment_moves.trailer_id IS 'Foreign key referencing the trailer that is being used for the shipment move';
COMMENT ON COLUMN shipment_moves.primary_worker_id IS 'Foreign key referencing the primary worker for the shipment move';
COMMENT ON COLUMN shipment_moves.secondary_worker_id IS 'Foreign key referencing the secondary worker for the shipment move';
COMMENT ON COLUMN shipment_moves.created_at IS 'Timestamp of when the shipment move was created, defaults to the current timestamp';
COMMENT ON COLUMN shipment_moves.updated_at IS 'Timestamp of the last update to the shipment move, defaults to the current timestamp';
