DROP INDEX IF EXISTS "idx_rate_quotes_margin_reporting";

--bun:split
DROP INDEX IF EXISTS "idx_carrier_assignments_rate_quote";

--bun:split
ALTER TABLE "carrier_assignments"
    DROP COLUMN IF EXISTS "rate_quote_id";
