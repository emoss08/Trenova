DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'shipment_status_enum') THEN
            CREATE TYPE shipment_status_enum AS ENUM ('New', 'InProgress', 'Completed', 'Hold', 'Billed', 'Voided');
        END IF;
    END
$$;

-- bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'rating_method_enum') THEN
            CREATE TYPE rating_method_enum AS ENUM ('FlatRate', 'PerMile', 'PerHundredWeight', 'PerStop', 'PerPound', 'Other');
        END IF;
    END
$$;

-- bun:split

CREATE TABLE
    IF NOT EXISTS "shipments"
(
    "id"                          uuid                 NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"            uuid                 NOT NULL,
    "organization_id"             uuid                 NOT NULL,
    "status"                      shipment_status_enum NOT NULL DEFAULT 'New',
    "pro_number"                  VARCHAR(20)          NOT NULL,
    "shipment_type_id"            uuid                 NOT NULL,
    "revenue_code_id"             uuid,
    "service_type_id"             uuid,
    "rating_unit"                 integer,
    "rating_method"               rating_method_enum,
    "other_charge_amount"         numeric(19, 4)       NOT NULL DEFAULT 0,
    "freight_charge_amount"       numeric(19, 4)       NOT NULL DEFAULT 0,
    "total_charge_amount"         numeric(19, 4)       NOT NULL DEFAULT 0,
    "pieces"                      numeric(10, 2)       NOT NULL DEFAULT 0,
    "weight"                      numeric(10, 2)       NOT NULL DEFAULT 0,
    "ready_to_bill"               boolean              NOT NULL DEFAULT false,
    "bill_date"                   date,
    "ship_date"                   date,
    "billed"                      boolean              NOT NULL DEFAULT false,
    "transferred_to_billing"      boolean              NOT NULL DEFAULT false,
    "transferred_to_billing_date" date,
    "trailer_type_id"             uuid,
    "tractor_type_id"             uuid,
    "temperature_min"             integer,
    "temperature_max"             integer,
    "bill_of_lading"              VARCHAR(20),
    "voided_comment"              TEXT,
    "auto_rated"                  boolean              NOT NULL DEFAULT false,
    "entry_method"                VARCHAR(20),
    "created_by_id"               uuid,
    "is_hazardous"                boolean              NOT NULL DEFAULT false,
    "estimated_delivery_date"     date,
    "actual_delivery_date"        date,
    "origin_location_id"          uuid,
    "destination_location_id"     uuid,
    "customer_id"                 uuid,
    "priority"                    integer,
    "special_instructions"        TEXT,
    "tracking_number"             VARCHAR(50),
    "total_distance"              numeric(10, 2),
    "created_at"                  TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "updated_at"                  TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("shipment_type_id") REFERENCES shipment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("revenue_code_id") REFERENCES revenue_codes ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("service_type_id") REFERENCES shipment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("tractor_type_id") REFERENCES equipment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("trailer_type_id") REFERENCES equipment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("created_by_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("origin_location_id") REFERENCES locations ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("destination_location_id") REFERENCES locations ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("customer_id") REFERENCES customers ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "shipments_pro_number_organization_id_unq" ON "shipments" (LOWER("pro_number"), organization_id);
CREATE INDEX IF NOT EXISTS "idx_shipments_status" ON "shipments" ("status");
CREATE INDEX IF NOT EXISTS "idx_shipments_ship_date" ON "shipments" ("ship_date");
CREATE INDEX IF NOT EXISTS "idx_shipments_bill_date" ON "shipments" ("bill_date");
CREATE INDEX IF NOT EXISTS "idx_shipments_customer_id" ON "shipments" ("customer_id");
CREATE INDEX IF NOT EXISTS "idx_shipments_origin_location_id" ON "shipments" ("origin_location_id");
CREATE INDEX IF NOT EXISTS "idx_shipments_destination_location_id" ON "shipments" ("destination_location_id");
CREATE INDEX IF NOT EXISTS "idx_shipments_created_by_id" ON "shipments" ("created_by_id");
CREATE INDEX IF NOT EXISTS "idx_shipments_business_unit_id" ON "shipments" ("business_unit_id");

-- bun:split

COMMENT ON COLUMN shipments.id IS 'Unique identifier for the location, generated as a UUID';
COMMENT ON COLUMN shipments.business_unit_id IS 'Foreign key referencing the business unit to which this location belongs';
COMMENT ON COLUMN shipments.organization_id IS 'Foreign key referencing the organization to which this location belongs';
COMMENT ON COLUMN shipments.status IS 'The current status of the shipment, using the shipment_status_enum (e.g., New, InProgress, Completed)';
COMMENT ON COLUMN shipments.shipment_type_id IS 'Foreign key referencing the type of shipment';
COMMENT ON COLUMN shipments.revenue_code_id IS 'Foreign key referencing the revenue code associated with this shipment';
COMMENT ON COLUMN shipments.service_type_id IS 'Foreign key referencing the service type associated with this shipment';
COMMENT ON COLUMN shipments.rating_unit IS 'The unit used for rating the shipment, represented as an integer';
COMMENT ON COLUMN shipments.rating_method IS 'The method used for rating the shipment, using the rating_method_enum (e.g., FlatRate, PerMile)';
COMMENT ON COLUMN shipments.other_charge_amount IS 'The amount of any additional charges associated with the shipment, with a default value of 0';
COMMENT ON COLUMN shipments.freight_charge_amount IS 'The amount charged for the freight, with a default value of 0';
COMMENT ON COLUMN shipments.total_charge_amount IS 'The total amount charged for the shipment, including all other charges, with a default value of 0';
COMMENT ON COLUMN shipments.pieces IS 'The number of pieces included in the shipment, with a default value of 0';
COMMENT ON COLUMN shipments.weight IS 'The total weight of the shipment, with a default value of 0';
COMMENT ON COLUMN shipments.ready_to_bill IS 'Indicates whether the shipment is ready to be billed, with a default value of false';
COMMENT ON COLUMN shipments.bill_date IS 'The date on which the shipment is billed';
COMMENT ON COLUMN shipments.ship_date IS 'The date on which the shipment is shipped';
COMMENT ON COLUMN shipments.billed IS 'Indicates whether the shipment has been billed, with a default value of false';
COMMENT ON COLUMN shipments.transferred_to_billing IS 'Indicates whether the shipment has been transferred to billing, with a default value of false';
COMMENT ON COLUMN shipments.transferred_to_billing_date IS 'The date on which the shipment was transferred to billing';
COMMENT ON COLUMN shipments.trailer_type_id IS 'Foreign key referencing the type of trailer used for the shipment';
COMMENT ON COLUMN shipments.tractor_type_id IS 'Foreign key referencing the type of tractor used for the shipment';
COMMENT ON COLUMN shipments.temperature_min IS 'The minimum temperature required for the shipment, if applicable';
COMMENT ON COLUMN shipments.temperature_max IS 'The maximum temperature required for the shipment, if applicable';
COMMENT ON COLUMN shipments.bill_of_lading IS 'The bill of lading number associated with the shipment, limited to 20 characters';
COMMENT ON COLUMN shipments.voided_comment IS 'Comment or reason for why the shipment was voided, if applicable';
COMMENT ON COLUMN shipments.auto_rated IS 'Indicates whether the shipment was rated automatically, with a default value of false';
COMMENT ON COLUMN shipments.entry_method IS 'The method used to enter the shipment into the system, limited to 20 characters';
COMMENT ON COLUMN shipments.created_by_id IS 'Foreign key referencing the user who created the shipment';
COMMENT ON COLUMN shipments.is_hazardous IS 'Indicates whether the shipment contains hazardous materials, with a default value of false';
COMMENT ON COLUMN shipments.estimated_delivery_date IS 'The estimated date for delivery of the shipment';
COMMENT ON COLUMN shipments.actual_delivery_date IS 'The actual date on which the shipment was delivered';
COMMENT ON COLUMN shipments.origin_location_id IS 'Foreign key referencing the origin location of the shipment';
COMMENT ON COLUMN shipments.destination_location_id IS 'Foreign key referencing the destination location of the shipment';
COMMENT ON COLUMN shipments.customer_id IS 'Foreign key referencing the customer associated with the shipment';
COMMENT ON COLUMN shipments.priority IS 'The priority level of the shipment, represented as an integer';
COMMENT ON COLUMN shipments.special_instructions IS 'Any special instructions associated with the shipment';
COMMENT ON COLUMN shipments.tracking_number IS 'The tracking number for the shipment, limited to 50 characters';
COMMENT ON COLUMN shipments.total_distance IS 'The total distance covered by the shipment, represented as a numeric value';
COMMENT ON COLUMN shipments.created_at IS 'Timestamp of when the shipment was created, defaults to the current timestamp';
COMMENT ON COLUMN shipments.updated_at IS 'Timestamp of the last update to the shipment, defaults to the current timestamp';
