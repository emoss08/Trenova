SET statement_timeout = 0;

DROP TABLE IF EXISTS "customer_billing_profile_document_types" CASCADE;

--bun:split
DROP TABLE IF EXISTS "customer_billing_profiles" CASCADE;

--bun:split
DROP TYPE IF EXISTS invoice_number_format_enum;

--bun:split
DROP TYPE IF EXISTS consolidation_group_by_enum;

--bun:split
DROP TYPE IF EXISTS invoice_method_enum;

--bun:split
DROP TYPE IF EXISTS credit_status_enum;

--bun:split
DROP TYPE IF EXISTS billing_cycle_type_enum;
