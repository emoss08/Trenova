-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260318134500_align_customer_profile_schema.tx.up.sql

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_bu_org" ON "customer_billing_profiles" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_created_updated" ON "customer_billing_profiles" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_billing_profile" ON "customer_billing_profile_document_types" ("billing_profile_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_document_type" ON "customer_billing_profile_document_types" ("document_type_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_bu_org" ON "customer_email_profiles" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_created_updated" ON "customer_email_profiles" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_from_email" ON "customer_email_profiles" ("from_email", "organization_id", "business_unit_id")WHERE "from_email" IS NOT NULL;
