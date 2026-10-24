-- One customer payment reconciles one deposit. Matching takes a row lock and
-- refuses a payment already matched elsewhere; this index makes the database
-- refuse it too. Data that already breaks the rule is left for a person to
-- untangle rather than rewritten here, so the index is only created once it
-- can hold.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM "bank_receipts"
        WHERE "matched_customer_payment_id" IS NOT NULL
        GROUP BY "organization_id", "business_unit_id", "matched_customer_payment_id"
        HAVING COUNT(*) > 1
    ) THEN
        CREATE UNIQUE INDEX IF NOT EXISTS "uq_bank_receipts_matched_payment"
            ON "bank_receipts" ("organization_id", "business_unit_id", "matched_customer_payment_id")
            WHERE "matched_customer_payment_id" IS NOT NULL;
    ELSE
        RAISE WARNING 'bank_receipts has payments matched to more than one deposit; uq_bank_receipts_matched_payment was not created';
    END IF;
END
$$;
