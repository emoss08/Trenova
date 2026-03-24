ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "count_detention_only_on_appointment_stops",
    DROP COLUMN IF EXISTS "count_late_only_on_appointment_stops";

--bun:split
ALTER TABLE "stops"
    DROP CONSTRAINT IF EXISTS "chk_stops_scheduled_window";

--bun:split
ALTER TABLE "stops"
    ADD CONSTRAINT "chk_stops_planned_times"
    CHECK ("scheduled_window_end" >= "scheduled_window_start");

--bun:split
ALTER TABLE "stops"
    ALTER COLUMN "scheduled_window_end" SET NOT NULL;

--bun:split
ALTER TABLE "stops"
    DROP COLUMN IF EXISTS "count_detention_override",
    DROP COLUMN IF EXISTS "count_late_override",
    DROP COLUMN IF EXISTS "schedule_type";

--bun:split
ALTER TABLE "stops"
    RENAME COLUMN "scheduled_window_end" TO "planned_departure";

--bun:split
ALTER TABLE "stops"
    RENAME COLUMN "scheduled_window_start" TO "planned_arrival";

--bun:split
ALTER INDEX IF EXISTS "idx_stops_shipment_move_schedule"
    RENAME TO "idx_stops_shipment_move";

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'stop_schedule_type_enum'
    ) THEN
        DROP TYPE "stop_schedule_type_enum";
    END IF;
END $$;
