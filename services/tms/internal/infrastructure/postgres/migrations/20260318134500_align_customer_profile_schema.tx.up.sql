ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "fuel_surcharge_method",
    DROP COLUMN IF EXISTS "use_factoring";

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'fuel_surcharge_method_enum'
    ) THEN
        DROP TYPE "fuel_surcharge_method_enum";
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_customer_billing_profiles_billing_cycle_day_of_week'
    ) THEN
        ALTER TABLE "customer_billing_profiles"
            ADD CONSTRAINT "ck_customer_billing_profiles_billing_cycle_day_of_week"
            CHECK ("billing_cycle_day_of_week" IS NULL OR ("billing_cycle_day_of_week" >= 0 AND "billing_cycle_day_of_week" <= 6));
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_customer_billing_profiles_invoice_copies'
    ) THEN
        ALTER TABLE "customer_billing_profiles"
            ADD CONSTRAINT "ck_customer_billing_profiles_invoice_copies"
            CHECK ("invoice_copies" >= 1);
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_customer_billing_profiles_grace_period_days'
    ) THEN
        ALTER TABLE "customer_billing_profiles"
            ADD CONSTRAINT "ck_customer_billing_profiles_grace_period_days"
            CHECK ("grace_period_days" >= 0);
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_customer_billing_profiles_detention_free_minutes'
    ) THEN
        ALTER TABLE "customer_billing_profiles"
            ADD CONSTRAINT "ck_customer_billing_profiles_detention_free_minutes"
            CHECK ("detention_free_minutes" >= 0);
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_customer_billing_profiles_billing_currency'
    ) THEN
        ALTER TABLE "customer_billing_profiles"
            ADD CONSTRAINT "ck_customer_billing_profiles_billing_currency"
            CHECK (char_length("billing_currency") = 3);
    END IF;
END $$;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_bu_org" ON "customer_billing_profiles"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profiles_created_updated" ON "customer_billing_profiles"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_billing_profile" ON "customer_billing_profile_document_types"("billing_profile_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_billing_profile_document_types_document_type" ON "customer_billing_profile_document_types"("document_type_id", "organization_id", "business_unit_id");

--bun:split
ALTER TABLE "customer_billing_profiles"
    ALTER COLUMN "payment_term" SET STATISTICS 1000;

--bun:split
ALTER TABLE "customer_email_profiles"
    ALTER COLUMN "subject" TYPE varchar(255);

--bun:split
ALTER TABLE "customer_email_profiles"
    ADD COLUMN IF NOT EXISTS "to_recipients" text,
    ADD COLUMN IF NOT EXISTS "cc_recipients" text,
    ADD COLUMN IF NOT EXISTS "bcc_recipients" text,
    ADD COLUMN IF NOT EXISTS "send_invoice_on_generation" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "include_shipment_detail" boolean NOT NULL DEFAULT FALSE;

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'customer_email_profiles'
          AND column_name = 'blind_copy'
    ) THEN
        EXECUTE '
            UPDATE "customer_email_profiles"
            SET "bcc_recipients" = "blind_copy"
            WHERE "bcc_recipients" IS NULL
              AND "blind_copy" IS NOT NULL
        ';
    END IF;
END $$;

--bun:split
ALTER TABLE "customer_email_profiles"
    DROP COLUMN IF EXISTS "blind_copy";

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_customer_email_profiles_id_org_bu'
    ) THEN
        ALTER TABLE "customer_email_profiles"
            ADD CONSTRAINT "uq_customer_email_profiles_id_org_bu"
            UNIQUE ("id", "organization_id", "business_unit_id");
    END IF;
END $$;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_bu_org" ON "customer_email_profiles"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_created_updated" ON "customer_email_profiles"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_from_email" ON "customer_email_profiles"("from_email", "organization_id", "business_unit_id")
    WHERE "from_email" IS NOT NULL;

--bun:split
ALTER TABLE "customer_email_profiles"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "customer_email_profiles"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;
