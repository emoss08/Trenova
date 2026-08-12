DROP TABLE IF EXISTS "rate_confirmation_tokens";

--bun:split
ALTER TABLE "rate_confirmations"
    DROP COLUMN IF EXISTS "generated_via",
    DROP COLUMN IF EXISTS "source_tender_offer_id",
    DROP COLUMN IF EXISTS "confirmed_via",
    DROP COLUMN IF EXISTS "confirmed_by_title";
