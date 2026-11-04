-- A facility's scorecard and its location desk card read the same window as detention
-- intelligence, narrowed to one location. The settled-window index orders a tenant's occurrences
-- by clock start alone, so narrowing to a facility scanned the tenant's whole window; this one
-- reaches a facility's window directly. The predicate is written as keel renders it (`IS FALSE`).
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_location_window"
    ON "detention_occurrences"("organization_id", "business_unit_id", "location_id", "clock_start_at")
    WHERE "is_open" IS FALSE;
