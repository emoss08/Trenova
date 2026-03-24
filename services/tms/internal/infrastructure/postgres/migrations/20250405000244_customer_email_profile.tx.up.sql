--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "customer_email_profiles"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Core fields
    "subject" varchar(255),
    "comment" text,
    "from_email" varchar(255),
    "to_recipients" text,
    "cc_recipients" text,
    "bcc_recipients" text,
    "read_receipt" boolean NOT NULL DEFAULT FALSE,
    "attachment_name" varchar(255),
    "send_invoice_on_generation" boolean NOT NULL DEFAULT TRUE,
    "include_shipment_detail" boolean NOT NULL DEFAULT FALSE,
    -- Metadata and versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_customer_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id", "customer_id"),
    CONSTRAINT "fk_customer_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one email profile per customer
    CONSTRAINT "uq_customer_email_profiles_customer" UNIQUE ("customer_id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_customer_email_profiles_id_org_bu" UNIQUE ("id", "organization_id", "business_unit_id")
);

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_customer_id" ON "customer_email_profiles"("customer_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_bu_org" ON "customer_email_profiles"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_created_updated" ON "customer_email_profiles"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_from_email" ON "customer_email_profiles"("from_email", "organization_id", "business_unit_id")
    WHERE "from_email" IS NOT NULL;

--bun:split
ALTER TABLE customer_email_profiles
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE customer_email_profiles
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
COMMENT ON TABLE "customer_email_profiles" IS 'Stores tenant-scoped invoice email delivery preferences for a customer';
