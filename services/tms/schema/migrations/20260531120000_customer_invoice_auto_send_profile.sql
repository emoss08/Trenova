ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "auto_send_invoice_on_generation" boolean NOT NULL DEFAULT TRUE;

DO $$
DECLARE
    has_billing_old boolean;
    has_email_old boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'customer_billing_profiles'
          AND column_name = 'summary_transmit_on_generation'
    ) INTO has_billing_old;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'customer_email_profiles'
          AND column_name = 'send_invoice_on_generation'
    ) INTO has_email_old;

    IF has_billing_old AND has_email_old THEN
        EXECUTE '
            UPDATE customer_billing_profiles AS cbp
            SET auto_send_invoice_on_generation =
                cbp.summary_transmit_on_generation AND COALESCE(cem.send_invoice_on_generation, FALSE)
            FROM customer_email_profiles AS cem
            WHERE cem.customer_id = cbp.customer_id
              AND cem.organization_id = cbp.organization_id
              AND cem.business_unit_id = cbp.business_unit_id';

        EXECUTE '
            UPDATE customer_billing_profiles AS cbp
            SET auto_send_invoice_on_generation = FALSE
            WHERE cbp.summary_transmit_on_generation = FALSE
               OR NOT EXISTS (
                   SELECT 1
                   FROM customer_email_profiles AS cem
                   WHERE cem.customer_id = cbp.customer_id
                     AND cem.organization_id = cbp.organization_id
                     AND cem.business_unit_id = cbp.business_unit_id
               )';
    ELSIF has_billing_old THEN
        EXECUTE '
            UPDATE customer_billing_profiles
            SET auto_send_invoice_on_generation = summary_transmit_on_generation';
    ELSIF has_email_old THEN
        EXECUTE '
            UPDATE customer_billing_profiles AS cbp
            SET auto_send_invoice_on_generation = COALESCE(cem.send_invoice_on_generation, FALSE)
            FROM customer_email_profiles AS cem
            WHERE cem.customer_id = cbp.customer_id
              AND cem.organization_id = cbp.organization_id
              AND cem.business_unit_id = cbp.business_unit_id';
    END IF;
END $$;

ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "summary_transmit_on_generation";

ALTER TABLE "customer_email_profiles"
    DROP COLUMN IF EXISTS "send_invoice_on_generation";
