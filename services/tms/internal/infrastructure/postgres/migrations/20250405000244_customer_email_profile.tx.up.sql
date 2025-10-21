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
    "subject" varchar(100),
    "comment" text,
    "from_email" varchar(255),
    "blind_copy" varchar(255), -- Comma separated list of email addresses
    "read_receipt" boolean NOT NULL DEFAULT FALSE,
    "attachment_name" varchar(255),
    -- Metadata and versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_customer_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id", "customer_id"),
    CONSTRAINT "fk_customer_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one billing profile per customer
    CONSTRAINT "uq_customer_email_profiles_customer" UNIQUE ("customer_id", "organization_id", "business_unit_id")
);

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_customer_id" ON "customer_email_profiles"("customer_id", "organization_id", "business_unit_id");

