ALTER TABLE "shipment_moves"
    ADD COLUMN IF NOT EXISTS "coverage_type" varchar(20) NOT NULL DEFAULT 'driver';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_moves_coverage_type" ON "shipment_moves"("coverage_type", "organization_id")
WHERE
    "coverage_type" <> 'driver';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_assignments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'Pending',
    "rate_method" varchar(50) NOT NULL DEFAULT 'Flat',
    "base_rate" numeric(19, 4) NOT NULL DEFAULT 0,
    "base_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "fuel_surcharge" numeric(19, 4) NOT NULL DEFAULT 0,
    "accessorial_total" numeric(19, 4) NOT NULL DEFAULT 0,
    "total_cost" numeric(19, 4) NOT NULL DEFAULT 0,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "pro_number" varchar(50),
    "external_driver_name" varchar(255),
    "external_driver_phone" varchar(20),
    "external_tractor_number" varchar(50),
    "external_trailer_number" varchar(50),
    "assigned_by_id" varchar(100),
    "confirmed_at" bigint,
    "canceled_at" bigint,
    "cancellation_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_assignments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignments_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_carrier_assignments_amounts" CHECK ("base_rate" >= 0 AND "fuel_surcharge" >= 0 AND "accessorial_total" >= 0 AND "total_cost" >= 0)
);

--bun:split
CREATE UNIQUE INDEX "idx_carrier_assignments_active_move" ON "carrier_assignments"("shipment_move_id", "organization_id", "business_unit_id")
WHERE
    "status" <> 'Canceled';

--bun:split
CREATE INDEX "idx_carrier_assignments_carrier" ON "carrier_assignments"("carrier_id", "organization_id");

--bun:split
CREATE INDEX "idx_carrier_assignments_status" ON "carrier_assignments"("status", "organization_id");

--bun:split
CREATE INDEX "idx_carrier_assignments_pro_number" ON "carrier_assignments"("pro_number", "organization_id")
WHERE
    "pro_number" IS NOT NULL;

--bun:split
COMMENT ON TABLE "carrier_assignments" IS 'Covers a shipment move with an external carrier and captures the negotiated buy rate';

--bun:split
CREATE TABLE IF NOT EXISTS "carrier_assignment_accessorials"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_assignment_id" varchar(100) NOT NULL,
    "accessorial_charge_id" varchar(100),
    "description" varchar(255) NOT NULL,
    "amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_assignment_accessorials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_assignment_accessorials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignment_accessorials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_assignment_accessorials_assignment" FOREIGN KEY ("carrier_assignment_id", "organization_id", "business_unit_id") REFERENCES "carrier_assignments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_assignment_accessorials_amount" CHECK ("amount" >= 0)
);

--bun:split
CREATE INDEX "idx_carrier_assignment_accessorials_assignment" ON "carrier_assignment_accessorials"("carrier_assignment_id", "organization_id");

--bun:split
COMMENT ON TABLE "carrier_assignment_accessorials" IS 'Accessorial cost lines attached to a carrier assignment buy rate';
