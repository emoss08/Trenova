--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Order-level charges (customs brokerage, order-wide fuel, etc.) that are not
-- attributable to a single leg. They roll into the order total and appear as their
-- own lines on a grouped invoice.
CREATE TABLE IF NOT EXISTS "order_charges"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "order_id" varchar(100) NOT NULL,
    "description" varchar(255) NOT NULL,
    "amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_order_charges" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_order_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_order_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_order_charges_order" FOREIGN KEY ("order_id", "organization_id", "business_unit_id") REFERENCES "orders"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_order_charges_order ON order_charges("order_id");

CREATE INDEX IF NOT EXISTS idx_order_charges_business_unit ON order_charges("business_unit_id");

CREATE INDEX IF NOT EXISTS idx_order_charges_organization ON order_charges("organization_id");

COMMENT ON TABLE "order_charges" IS 'Order-level charges not attributable to a single shipment leg';
