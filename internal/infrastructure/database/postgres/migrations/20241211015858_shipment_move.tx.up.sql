-- Enum with documentation for each status
CREATE TYPE "move_status_enum" AS ENUM(
    'New', -- Initial state when move is created
    'Assigned', -- Move is currently being executed
    'InTransit', -- Move is currently being executed
    'Completed', -- Move has been completed successfully
    'Canceled' -- Move has been cancelled and won't be completed
);

CREATE TABLE IF NOT EXISTS "shipment_moves"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Primary keys
    "shipment_id" varchar(100) NOT NULL,
    -- Core Fields
    "status" move_status_enum NOT NULL DEFAULT 'New',
    "loaded" boolean NOT NULL DEFAULT TRUE,
    "sequence" integer NOT NULL DEFAULT 0 CHECK ("sequence" >= 0),
    "distance" float,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipment_moves" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_moves_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_moves_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_moves_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_created_at" ON "shipment_moves"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_status" ON "shipment_moves"("status");

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_business_unit" ON "shipment_moves"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_shipment" ON "shipment_moves"("shipment_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_sequence" ON "shipment_moves"("sequence");

COMMENT ON TABLE shipment_moves IS 'Stores information about individual moves within a shipment journey';

