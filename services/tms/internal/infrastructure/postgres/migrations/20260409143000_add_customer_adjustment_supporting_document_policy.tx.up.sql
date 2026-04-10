DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'invoice_adjustment_supporting_document_policy_enum'
    ) THEN
        CREATE TYPE invoice_adjustment_supporting_document_policy_enum AS ENUM (
            'Inherit',
            'Required',
            'Optional'
        );
    END IF;
END $$;

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "invoice_adjustment_supporting_document_policy"
    invoice_adjustment_supporting_document_policy_enum NOT NULL DEFAULT 'Inherit';
