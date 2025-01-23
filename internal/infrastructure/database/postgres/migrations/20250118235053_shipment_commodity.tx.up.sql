CREATE TABLE IF NOT EXISTS "shipment_commodities"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "commodity_id" varchar(100) NOT NULL,
    -- Core fields
    "weight" float NOT NULL,
    "pieces" int NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_shipment_commodities" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id", "commodity_id"),
    CONSTRAINT "fk_shipment_commodities_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_commodities_commodity" FOREIGN KEY ("commodity_id", "business_unit_id", "organization_id") REFERENCES "commodities"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for shipment commodities table
CREATE INDEX "idx_shipment_commodities_shipment" ON "shipment_commodities"("shipment_id");

CREATE INDEX "idx_shipment_commodities_commodity" ON "shipment_commodities"("commodity_id");

CREATE INDEX "idx_shipment_commodities_business_unit" ON "shipment_commodities"("business_unit_id");

CREATE INDEX "idx_shipment_commodities_organization" ON "shipment_commodities"("organization_id");

CREATE INDEX "idx_shipment_commodities_created_updated" ON "shipment_commodities"("created_at", "updated_at");

COMMENT ON TABLE "shipment_commodities" IS 'Stores information about shipment commodities';

