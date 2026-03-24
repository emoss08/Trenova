DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'stop_schedule_type_enum'
    ) THEN
        CREATE TYPE "stop_schedule_type_enum" AS ENUM ('Open', 'Appointment');
    END IF;
END $$;

--bun:split
ALTER TABLE "stops"
    RENAME COLUMN "planned_arrival" TO "scheduled_window_start";

--bun:split
ALTER TABLE "stops"
    RENAME COLUMN "planned_departure" TO "scheduled_window_end";

--bun:split
ALTER TABLE "stops"
    ADD COLUMN IF NOT EXISTS "schedule_type" stop_schedule_type_enum NOT NULL DEFAULT 'Open',
    ADD COLUMN IF NOT EXISTS "count_late_override" boolean,
    ADD COLUMN IF NOT EXISTS "count_detention_override" boolean;

--bun:split
ALTER TABLE "stops"
    ALTER COLUMN "scheduled_window_end" DROP NOT NULL;

--bun:split
ALTER TABLE "stops"
    DROP CONSTRAINT IF EXISTS "chk_stops_planned_times";

--bun:split
ALTER TABLE "stops"
    ADD CONSTRAINT "chk_stops_scheduled_window"
    CHECK (
        "scheduled_window_start" > 0
        AND (
            "scheduled_window_end" IS NULL
            OR "scheduled_window_end" >= "scheduled_window_start"
        )
    );

--bun:split
ALTER INDEX IF EXISTS "idx_stops_shipment_move"
    RENAME TO "idx_stops_shipment_move_schedule";

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "count_late_only_on_appointment_stops" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "count_detention_only_on_appointment_stops" boolean NOT NULL DEFAULT FALSE;
