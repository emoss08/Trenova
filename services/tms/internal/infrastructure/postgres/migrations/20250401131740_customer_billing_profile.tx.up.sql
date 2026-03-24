CREATE TYPE "billing_cycle_type_enum" AS ENUM(
    'Immediate',
    'Daily',
    'Weekly',
    'BiWeekly',
    'Monthly',
    'Quarterly',
    'PerShipment'
);

--bun:split
CREATE TYPE "credit_status_enum" AS ENUM(
    'Active',
    'Warning',
    'Hold',
    'Suspended',
    'Review'
);

--bun:split
CREATE TYPE "invoice_method_enum" AS ENUM(
    'Individual',
    'Summary',
    'SummaryWithDetail'
);

--bun:split
CREATE TYPE "consolidation_group_by_enum" AS ENUM(
    'None',
    'Location',
    'PONumber',
    'BOL',
    'Division'
);

--bun:split
CREATE TYPE "invoice_number_format_enum" AS ENUM(
    'Default',
    'CustomPrefix',
    'POBased'
);

--bun:split
CREATE TABLE IF NOT EXISTS "customer_billing_profiles"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Billing Cycle
    "billing_cycle_type" billing_cycle_type_enum NOT NULL DEFAULT 'Immediate',
    "billing_cycle_day_of_week" smallint,
    "payment_term" payment_term_enum NOT NULL DEFAULT 'Net30',
    -- Billing Control Overrides
    "has_billing_control_overrides" boolean NOT NULL DEFAULT FALSE,
    -- Credit Management
    "credit_limit" numeric(12, 2),
    "credit_balance" numeric(12, 2) NOT NULL DEFAULT 0,
    "credit_status" credit_status_enum NOT NULL DEFAULT 'Active',
    "enforce_credit_limit" boolean NOT NULL DEFAULT FALSE,
    "auto_credit_hold" boolean NOT NULL DEFAULT FALSE,
    "credit_hold_reason" text,
    -- Invoice Settings
    "invoice_method" invoice_method_enum NOT NULL DEFAULT 'Individual',
    "summary_transmit_on_generation" boolean NOT NULL DEFAULT TRUE,
    "allow_invoice_consolidation" boolean NOT NULL DEFAULT FALSE,
    "consolidation_period_days" integer NOT NULL DEFAULT 7 CHECK ("consolidation_period_days" >= 1),
    "consolidation_group_by" consolidation_group_by_enum NOT NULL DEFAULT 'None',
    "invoice_number_format" invoice_number_format_enum NOT NULL DEFAULT 'Default',
    "customer_invoice_prefix" varchar(20),
    "invoice_copies" smallint NOT NULL DEFAULT 1,
    -- Account References
    "revenue_account_id" varchar(100),
    "ar_account_id" varchar(100),
    -- Late Charges
    "apply_late_charges" boolean NOT NULL DEFAULT FALSE,
    "late_charge_rate" numeric(5, 2),
    "grace_period_days" smallint NOT NULL DEFAULT 0,
    -- Tax
    "tax_exempt" boolean NOT NULL DEFAULT FALSE,
    "tax_exempt_number" varchar(50),
    -- Billing Automation
    "enforce_customer_billing_req" boolean NOT NULL DEFAULT TRUE,
    "validate_customer_rates" boolean NOT NULL DEFAULT TRUE,
    "auto_transfer" boolean NOT NULL DEFAULT TRUE,
    "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT TRUE,
    "auto_bill" boolean NOT NULL DEFAULT TRUE,
    -- Detention
    "detention_billing_enabled" boolean NOT NULL DEFAULT FALSE,
    "detention_free_minutes" smallint NOT NULL DEFAULT 120,
    "detention_rate_per_hour" numeric(8, 2),
    -- Accessorials
    "auto_apply_accessorials" boolean NOT NULL DEFAULT TRUE,
    -- Currency
    "billing_currency" varchar(3) NOT NULL DEFAULT 'USD',
    -- Document Requirements
    "require_po_number" boolean NOT NULL DEFAULT FALSE,
    "require_bol_number" boolean NOT NULL DEFAULT FALSE,
    "require_delivery_number" boolean NOT NULL DEFAULT FALSE,
    -- Notes
    "billing_notes" text,
    -- Metadata and versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_customer_billing_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id", "customer_id"),
    CONSTRAINT "fk_customer_billing_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_customer_billing_profiles_customer" UNIQUE ("customer_id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_customer_billing_profiles_id_org_bu" UNIQUE ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "ck_customer_billing_profiles_billing_cycle_day_of_week" CHECK ("billing_cycle_day_of_week" IS NULL OR ("billing_cycle_day_of_week" >= 0 AND "billing_cycle_day_of_week" <= 6)),
    CONSTRAINT "ck_customer_billing_profiles_invoice_copies" CHECK ("invoice_copies" >= 1),
    CONSTRAINT "ck_customer_billing_profiles_grace_period_days" CHECK ("grace_period_days" >= 0),
    CONSTRAINT "ck_customer_billing_profiles_detention_free_minutes" CHECK ("detention_free_minutes" >= 0),
    CONSTRAINT "ck_customer_billing_profiles_billing_currency" CHECK (char_length("billing_currency") = 3)
);

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_customer_id" ON "customer_billing_profiles"("customer_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_bu_org" ON "customer_billing_profiles"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_created_updated" ON "customer_billing_profiles"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_credit_status" ON "customer_billing_profiles"("credit_status", "organization_id", "business_unit_id")
    WHERE "credit_status" IN ('Hold', 'Suspended', 'Warning');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_billing_cycle" ON "customer_billing_profiles"("billing_cycle_type", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_revenue_account" ON "customer_billing_profiles"("revenue_account_id", "organization_id", "business_unit_id")
    WHERE "revenue_account_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_ar_account" ON "customer_billing_profiles"("ar_account_id", "organization_id", "business_unit_id")
    WHERE "ar_account_id" IS NOT NULL;

--bun:split
ALTER TABLE customer_billing_profiles
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE customer_billing_profiles
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE customer_billing_profiles
    ALTER COLUMN credit_status SET STATISTICS 1000;

--bun:split
ALTER TABLE customer_billing_profiles
    ALTER COLUMN payment_term SET STATISTICS 1000;

--bun:split
CREATE TABLE IF NOT EXISTS "customer_billing_profile_document_types"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "billing_profile_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    CONSTRAINT "pk_customer_billing_profile_document_types" PRIMARY KEY ("organization_id", "business_unit_id", "billing_profile_id", "document_type_id"),
    CONSTRAINT "fk_customer_billing_profile_document_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_billing_profile" FOREIGN KEY ("billing_profile_id", "organization_id", "business_unit_id") REFERENCES "customer_billing_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_billing_profile" ON "customer_billing_profile_document_types"("billing_profile_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_document_type" ON "customer_billing_profile_document_types"("document_type_id", "organization_id", "business_unit_id");

--bun:split
ALTER TABLE customer_billing_profile_document_types
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE customer_billing_profile_document_types
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
COMMENT ON TABLE customer_billing_profiles IS 'Stores tenant-scoped billing controls and invoice preferences for a customer';

--bun:split
COMMENT ON COLUMN customer_billing_profiles.billing_cycle_day_of_week IS 'Day of week used when the billing cycle type requires a weekly cadence (0-6)';

--bun:split
COMMENT ON COLUMN customer_billing_profiles.credit_limit IS 'Maximum allowed outstanding balance for the customer';

--bun:split
COMMENT ON COLUMN customer_billing_profiles.credit_balance IS 'Current outstanding receivables balance for the customer';

--bun:split
COMMENT ON COLUMN customer_billing_profiles.billing_currency IS 'ISO 4217 billing currency code';

COMMENT ON TABLE customer_billing_profile_document_types IS 'Junction table linking billing profiles to their assigned document types';
