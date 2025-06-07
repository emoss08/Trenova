-- Enums
CREATE TYPE assignment_status_enum AS ENUM(
    'New', -- Initial state when assignment is created
    'InProgress', -- Assignment is accepted
    'Completed', -- Assignment is in transit
    'Canceled' -- Assignment is cancelled
);

--bun:split
CREATE TABLE IF NOT EXISTS "assignments"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Relationship identifiers (Non-Primary-Keys)
    "shipment_move_id" varchar(100) NOT NULL,
    "primary_worker_id" varchar(100) NOT NULL,
    "tractor_id" varchar(100) NOT NULL,
    "trailer_id" varchar(100),
    "secondary_worker_id" varchar(100),
    -- Core fields
    "status" assignment_status_enum NOT NULL DEFAULT 'New',
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_primary_worker" FOREIGN KEY ("primary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_secondary_worker" FOREIGN KEY ("secondary_worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_tractor" FOREIGN KEY ("tractor_id", "organization_id", "business_unit_id") REFERENCES "tractors"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_trailer" FOREIGN KEY ("trailer_id", "organization_id", "business_unit_id") REFERENCES "trailers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assignments_status" ON "assignments"("status");

CREATE INDEX IF NOT EXISTS "idx_assignments_created_at" ON "assignments"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_assignments_business_unit" ON "assignments"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_assignments_shipment_move" ON "assignments"("shipment_move_id", "organization_id");

-- Comments
COMMENT ON TABLE assignments IS 'Stores information about assignments for shipment moves';

