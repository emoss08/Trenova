CREATE TABLE IF NOT EXISTS "shipment_types"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(100) NOT NULL,
    "description" text,
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipment_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for shipment_types table
CREATE UNIQUE INDEX "idx_shipment_types_code" ON "shipment_types"(lower("code"), "organization_id");

CREATE INDEX "idx_shipment_types_business_unit" ON "shipment_types"("business_unit_id");

CREATE INDEX "idx_shipment_types_organization" ON "shipment_types"("organization_id");

CREATE INDEX "idx_shipment_types_created_updated" ON "shipment_types"("created_at", "updated_at");

COMMENT ON TABLE "shipment_types" IS 'Stores information about shipment types';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "shipment_type_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

