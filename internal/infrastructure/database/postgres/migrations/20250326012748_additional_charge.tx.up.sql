CREATE TABLE IF NOT EXISTS "additional_charges"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "accessorial_charge_id" varchar(100) NOT NULL,
    -- Core fields
    "unit" integer NOT NULL,
    "method" accessorial_method_enum NOT NULL,
    "amount" numeric(19, 4) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_additional_charges" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id", "accessorial_charge_id"),
    CONSTRAINT "fk_additional_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_additional_charges_accessorial_charge" FOREIGN KEY ("accessorial_charge_id", "business_unit_id", "organization_id") REFERENCES "accessorial_charges"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for additional charges table
CREATE INDEX "idx_additional_charges_shipment" ON "additional_charges"("shipment_id");

CREATE INDEX "idx_additional_charges_accessorial_charge" ON "additional_charges"("accessorial_charge_id");

CREATE INDEX "idx_additional_charges_business_unit" ON "additional_charges"("business_unit_id");

CREATE INDEX "idx_additional_charges_organization" ON "additional_charges"("organization_id");

CREATE INDEX "idx_additional_charges_created_updated" ON "additional_charges"("created_at", "updated_at");

COMMENT ON TABLE "additional_charges" IS 'Stores information about additional charges';

