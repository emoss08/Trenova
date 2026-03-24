ALTER TABLE "customer_email_profiles"
    ADD COLUMN IF NOT EXISTS "blind_copy" varchar(255);

--bun:split
UPDATE "customer_email_profiles"
SET "blind_copy" = "bcc_recipients"
WHERE "blind_copy" IS NULL;

--bun:split
DROP INDEX IF EXISTS "idx_customer_email_profiles_from_email";

--bun:split
DROP INDEX IF EXISTS "idx_customer_email_profiles_created_updated";

--bun:split
DROP INDEX IF EXISTS "idx_customer_email_profiles_bu_org";

--bun:split
ALTER TABLE "customer_email_profiles"
    DROP CONSTRAINT IF EXISTS "uq_customer_email_profiles_id_org_bu";

--bun:split
ALTER TABLE "customer_email_profiles"
    DROP COLUMN IF EXISTS "to_recipients",
    DROP COLUMN IF EXISTS "cc_recipients",
    DROP COLUMN IF EXISTS "bcc_recipients",
    DROP COLUMN IF EXISTS "send_invoice_on_generation",
    DROP COLUMN IF EXISTS "include_shipment_detail";

--bun:split
ALTER TABLE "customer_email_profiles"
    ALTER COLUMN "subject" TYPE varchar(100);

--bun:split
DROP INDEX IF EXISTS "idx_customer_billing_profile_document_types_document_type";

--bun:split
DROP INDEX IF EXISTS "idx_customer_billing_profile_document_types_billing_profile";

--bun:split
DROP INDEX IF EXISTS "idx_customer_billing_profiles_created_updated";

--bun:split
DROP INDEX IF EXISTS "idx_customer_billing_profiles_bu_org";

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP CONSTRAINT IF EXISTS "ck_customer_billing_profiles_billing_currency",
    DROP CONSTRAINT IF EXISTS "ck_customer_billing_profiles_detention_free_minutes",
    DROP CONSTRAINT IF EXISTS "ck_customer_billing_profiles_grace_period_days",
    DROP CONSTRAINT IF EXISTS "ck_customer_billing_profiles_invoice_copies",
    DROP CONSTRAINT IF EXISTS "ck_customer_billing_profiles_billing_cycle_day_of_week";

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'fuel_surcharge_method_enum'
    ) THEN
        CREATE TYPE "fuel_surcharge_method_enum" AS ENUM(
            'None',
            'Percentage',
            'PerMile',
            'FlatSchedule',
            'Included'
        );
    END IF;
END $$;

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_method" fuel_surcharge_method_enum NOT NULL DEFAULT 'None',
    ADD COLUMN IF NOT EXISTS "use_factoring" boolean NOT NULL DEFAULT FALSE;
