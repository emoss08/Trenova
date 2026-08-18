-- A carrier assignment records what the carrier is paid. Once a contract
-- decides that number rather than a person typing it, the assignment has to
-- name the quote it came from, or "why is this carrier being paid this" has no
-- answer on the buy side while the sell side has a full trace.
ALTER TABLE "carrier_assignments"
    ADD COLUMN IF NOT EXISTS "rate_quote_id" VARCHAR(100);

--bun:split
-- Partial, because most rows predate auto-rating and a quote is only ever
-- looked up from an assignment that has one.
CREATE INDEX IF NOT EXISTS "idx_carrier_assignments_rate_quote" ON "carrier_assignments"("organization_id", "business_unit_id", "rate_quote_id")
WHERE
    "rate_quote_id" IS NOT NULL;

--bun:split
-- Margin is measured once, when both sides of a load are known, and it belongs
-- on the sell side quote: that is the row that already carries the revenue, and
-- putting cost beside it makes margin by lane, customer or carrier a plain
-- query rather than a join nobody will write.
--
-- The columns already exist on rate_quotes. What was missing is a way to find
-- the quote a shipment is billed from quickly enough to update it on every
-- assignment, which this index gives.
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_margin_reporting" ON "rate_quotes"("organization_id", "business_unit_id", "party_type", "party_id", "rated_at")
WHERE
    "margin_percent" IS NOT NULL;
