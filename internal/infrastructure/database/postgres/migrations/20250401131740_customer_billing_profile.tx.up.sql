CREATE TYPE "billing_cycle_type_enum" AS ENUM(
    'Immediate',
    'Daily',
    'Weekly',
    'Monthly',
    'Quarterly'
);

CREATE TABLE IF NOT EXISTS "customer_billing_profiles"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Core fields
    "billing_cycle_type" billing_cycle_type_enum NOT NULL DEFAULT 'Immediate',
    "document_type_ids" varchar(100)[] NOT NULL DEFAULT '{}',
    -- Billing Control Overrides
    "enforce_customer_billing_req" boolean NOT NULL DEFAULT TRUE,
    "validate_customer_rates" boolean NOT NULL DEFAULT TRUE,
    "has_overrides" boolean NOT NULL DEFAULT FALSE,
    "payment_term" payment_term_enum NOT NULL DEFAULT 'Net30',
    "auto_transfer" boolean NOT NULL DEFAULT TRUE,
    "transfer_criteria" transfer_criteria_enum NOT NULL DEFAULT 'ReadyAndCompleted',
    "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT TRUE,
    "auto_bill" boolean NOT NULL DEFAULT TRUE,
    "auto_bill_criteria" auto_bill_criteria_enum NOT NULL DEFAULT 'Delivered',
    -- Metadata and versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_customer_billing_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id", "customer_id"),
    CONSTRAINT "fk_customer_billing_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one billing profile per customer
    CONSTRAINT "uq_customer_billing_profiles_customer" UNIQUE ("customer_id", "organization_id", "business_unit_id")
);

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_customer_id" ON "customer_billing_profiles"("customer_id", "organization_id", "business_unit_id");

-- bun:split
CREATE TABLE IF NOT EXISTS "billing_profile_document_types"(
    -- Primary identifiers
    "billing_profile_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL, -- Added this to match parent table's PK
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_billing_profile_document_types" PRIMARY KEY ("billing_profile_id", "document_type_id"),
    CONSTRAINT "fk_billing_profile_document_types_billing_profile" FOREIGN KEY ("billing_profile_id", "organization_id", "business_unit_id", "customer_id") REFERENCES "customer_billing_profiles"("id", "organization_id", "business_unit_id", "customer_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_profile_document_types_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

