DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adjustment_attachment_policy_enum') THEN
        CREATE TYPE adjustment_attachment_policy_enum AS ENUM ('Optional', 'RequiredForCreditOrWriteOff', 'RequiredForAll');
    END IF;
END $$;

--bun:split
ALTER TABLE "invoice_adjustment_controls"
    ADD COLUMN IF NOT EXISTS "adjustment_attachment_requirement" adjustment_attachment_policy_enum NOT NULL DEFAULT 'Optional';

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'invoice_adjustment_supporting_document_policy_enum') THEN
        CREATE TYPE invoice_adjustment_supporting_document_policy_enum AS ENUM ('Inherit', 'Required', 'Optional');
    END IF;
END $$;

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "invoice_adjustment_supporting_document_policy" invoice_adjustment_supporting_document_policy_enum NOT NULL DEFAULT 'Inherit';
