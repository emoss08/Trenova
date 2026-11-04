-- How the freight changes hands at a stop: live (the truck waits while the facility loads or
-- unloads it) or a drop (a trailer is left or taken and the tractor goes on). Detention copies it
-- onto each occurrence, so detention intelligence can show what live stops cost against dropped
-- ones without joining back to the stop.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'stop_handling_enum'
    ) THEN
        CREATE TYPE "stop_handling_enum" AS ENUM ('Live', 'Drop');
    END IF;
END $$;

ALTER TABLE "stops"
    ADD COLUMN IF NOT EXISTS "handling" stop_handling_enum NOT NULL DEFAULT 'Live';

ALTER TABLE "detention_occurrences"
    ADD COLUMN IF NOT EXISTS "handling" stop_handling_enum NOT NULL DEFAULT 'Live';

-- Detention intelligence reads every settled occurrence whose clock started inside a window,
-- for one tenant, and folds it into its figures. Without this index each read scanned the
-- tenant's whole occurrence history. The predicate is written as keel renders the query's
-- (`IS FALSE`), which Postgres does not prove from `= FALSE`.
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_settled_window"
    ON "detention_occurrences"("organization_id", "business_unit_id", "clock_start_at")
    WHERE "is_open" IS FALSE;
