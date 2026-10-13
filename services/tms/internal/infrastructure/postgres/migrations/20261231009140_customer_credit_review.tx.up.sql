-- A customer's credit standing is reviewed against its open receivables each
-- time its ledger moves. credit_hold_automatic marks a hold or warning the
-- review placed, so only the review lifts it; a status a person set stays
-- until a person changes it. credit_reviewed_at is when credit_balance, the
-- open receivables, was last computed.
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "credit_hold_automatic" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "credit_reviewed_at" BIGINT;
