-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250401131740_customer_billing_profile.tx.up.sql

CREATE TABLE IF NOT EXISTS "customer_billing_profiles"(
    "id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "billing_cycle_type" TEXT NOT NULL DEFAULT 'Immediate',
    "billing_cycle_day_of_week" INTEGER,
    "payment_term" TEXT NOT NULL DEFAULT 'Net30',
    "has_billing_control_overrides" INTEGER NOT NULL DEFAULT 0,
    "credit_limit" NUMERIC,
    "credit_balance" NUMERIC NOT NULL DEFAULT 0,
    "credit_status" TEXT NOT NULL DEFAULT 'Active',
    "enforce_credit_limit" INTEGER NOT NULL DEFAULT 0,
    "auto_credit_hold" INTEGER NOT NULL DEFAULT 0,
    "credit_hold_reason" TEXT,
    "invoice_method" TEXT NOT NULL DEFAULT 'Individual',
    "summary_transmit_on_generation" INTEGER NOT NULL DEFAULT 1,
    "allow_invoice_consolidation" INTEGER NOT NULL DEFAULT 0,
    "consolidation_period_days" INTEGER NOT NULL DEFAULT 7 CHECK ("consolidation_period_days" >= 1),
    "consolidation_group_by" TEXT NOT NULL DEFAULT 'None',
    "invoice_number_format" TEXT NOT NULL DEFAULT 'Default',
    "customer_invoice_prefix" TEXT,
    "invoice_copies" INTEGER NOT NULL DEFAULT 1,
    "revenue_account_id" TEXT,
    "ar_account_id" TEXT,
    "apply_late_charges" INTEGER NOT NULL DEFAULT 0,
    "late_charge_rate" NUMERIC,
    "grace_period_days" INTEGER NOT NULL DEFAULT 0,
    "tax_exempt" INTEGER NOT NULL DEFAULT 0,
    "tax_exempt_number" TEXT,
    "enforce_customer_billing_req" INTEGER NOT NULL DEFAULT 1,
    "validate_customer_rates" INTEGER NOT NULL DEFAULT 1,
    "auto_transfer" INTEGER NOT NULL DEFAULT 1,
    "auto_mark_ready_to_bill" INTEGER NOT NULL DEFAULT 1,
    "auto_bill" INTEGER NOT NULL DEFAULT 1,
    "detention_billing_enabled" INTEGER NOT NULL DEFAULT 0,
    "detention_free_minutes" INTEGER NOT NULL DEFAULT 120,
    "detention_rate_per_hour" NUMERIC,
    "auto_apply_accessorials" INTEGER NOT NULL DEFAULT 1,
    "billing_currency" TEXT NOT NULL DEFAULT 'USD',
    "require_po_number" INTEGER NOT NULL DEFAULT 0,
    "require_bol_number" INTEGER NOT NULL DEFAULT 0,
    "require_delivery_number" INTEGER NOT NULL DEFAULT 0,
    "billing_notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
    CONSTRAINT "ck_customer_billing_profiles_billing_currency" CHECK (length("billing_currency") = 3)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_customer_id" ON "customer_billing_profiles" ("customer_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_bu_org" ON "customer_billing_profiles" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_created_updated" ON "customer_billing_profiles" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_credit_status" ON "customer_billing_profiles" ("credit_status", "organization_id", "business_unit_id")WHERE "credit_status" IN ('Hold', 'Suspended', 'Warning');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_billing_cycle" ON "customer_billing_profiles" ("billing_cycle_type", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_revenue_account" ON "customer_billing_profiles" ("revenue_account_id", "organization_id", "business_unit_id")WHERE "revenue_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_ar_account" ON "customer_billing_profiles" ("ar_account_id", "organization_id", "business_unit_id")WHERE "ar_account_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "customer_billing_profile_document_types"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "billing_profile_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    CONSTRAINT "pk_customer_billing_profile_document_types" PRIMARY KEY ("organization_id", "business_unit_id", "billing_profile_id", "document_type_id"),
    CONSTRAINT "fk_customer_billing_profile_document_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_billing_profile" FOREIGN KEY ("billing_profile_id", "organization_id", "business_unit_id") REFERENCES "customer_billing_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_billing_profile_document_types_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_billing_profile" ON "customer_billing_profile_document_types" ("billing_profile_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_document_type" ON "customer_billing_profile_document_types" ("document_type_id", "organization_id", "business_unit_id");
