--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A year-end adjusting period deliberately shares its date range with the final
-- operating period, so it has to sit outside the no-overlap exclusion. Operating
-- periods still may not overlap each other.
ALTER TABLE "fiscal_periods"
    DROP CONSTRAINT IF EXISTS "no_overlapping_periods";

--bun:split
ALTER TABLE "fiscal_periods"
    ADD CONSTRAINT "no_overlapping_periods"
    EXCLUDE USING gist("organization_id" WITH =, int8range("start_date", "end_date", '[]') WITH &&)
    WHERE (NOT "is_adjusting");
