ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "invoice_adjustment_supporting_document_policy";

--bun:split
DROP TYPE IF EXISTS invoice_adjustment_supporting_document_policy_enum;
