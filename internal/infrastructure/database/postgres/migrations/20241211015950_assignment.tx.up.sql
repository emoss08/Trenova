-- Enums
CREATE TYPE assignment_status_enum AS ENUM(
    'Pending', -- Initial state when assignment is created
    'Accepted', -- Assignment is accepted
    'InTransit', -- Assignment is in transit
    'Completed', -- Assignment is completed
    'Cancelled', -- Assignment is cancelled
    'Exception' -- Assignment encountered an issue
);

CREATE TYPE assignment_type_enum AS ENUM(
    'Dedicated', -- Dedicated assignment
    'Pool', -- Pool assignment
    'Spot' -- Spot assignment
);

--bun:split
CREATE TABLE IF NOT EXISTS "assignments"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    -- Core fields
    "status" assignment_status_enum NOT NULL DEFAULT 'Pending',
    "type" assignment_type_enum NOT NULL DEFAULT 'Dedicated',
    "exception_note" text CHECK (("status" = 'Exception' AND "exception_note" IS NOT NULL) OR ("status" != 'Exception' AND "exception_note" IS NULL)),
    "notes" text,
    -- Metadata
    "created_by_id" varchar(100) NOT NULL,
    "updated_by_id" varchar(100),
    "change_reason" text,
    "version" bigint NOT NULL DEFAULT 1,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assignments_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assignments_status" ON "assignments"("status");

CREATE INDEX IF NOT EXISTS "idx_assignments_type" ON "assignments"("type");

CREATE INDEX IF NOT EXISTS "idx_assignments_created_at" ON "assignments"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_assignments_business_unit" ON "assignments"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_assignments_shipment_move" ON "assignments"("shipment_move_id", "organization_id");

-- Comments
COMMENT ON TABLE assignments IS 'Stores information about assignments for shipment moves';

