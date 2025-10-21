--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE billing_type AS ENUM(
    'Invoice',
    'CreditMemo',
    'DebitMemo'
);

--bun:split
CREATE TYPE billing_queue_status AS ENUM(
    'ReadyForReview',
    'InReview',
    'Approved',
    'Canceled',
    'Exception'
);

--bun:split
CREATE TABLE IF NOT EXISTS billing_queue_items(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "assigned_biller_id" varchar(100),
    -- Core Fields
    "status" billing_queue_status NOT NULL DEFAULT 'ReadyForReview',
    "bill_type" billing_type NOT NULL DEFAULT 'Invoice',
    "review_notes" text,
    "exception_notes" text,
    "review_started_at" bigint,
    "review_completed_at" bigint,
    -- Cancellation Related Fields
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "cancel_reason" varchar(100),
    -- Timestamps
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_billing_queue_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_queue_items_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_assigned_biller" FOREIGN KEY ("assigned_biller_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_billing_queue_items_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "ck_billing_queue_items_status" CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception')),
    CONSTRAINT "ck_billing_queue_items_exception_notes" CHECK ("status" != 'Exception' OR "exception_notes" IS NOT NULL)
);

--bun:split
-- Add unique constraint to ensure a shipment can't have multiple entries of the same bill_type
CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_queue_items_shipment_bill_type 
ON billing_queue_items("shipment_id", "organization_id", "business_unit_id", "bill_type");

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_shipment_id ON billing_queue_items("shipment_id");

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_organization_id ON billing_queue_items("organization_id");


--bun:split
COMMENT ON TABLE billing_queue_items IS 'Stores billing queue items for billing';
