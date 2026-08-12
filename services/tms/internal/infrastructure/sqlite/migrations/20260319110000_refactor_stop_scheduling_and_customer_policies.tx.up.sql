-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260319110000_refactor_stop_scheduling_and_customer_policies.tx.up.sql

ALTER TABLE "stops" RENAME COLUMN "planned_arrival" TO "scheduled_window_start";

--bun:split

ALTER TABLE "stops" RENAME COLUMN "planned_departure" TO "scheduled_window_end";

--bun:split

ALTER TABLE "stops" ADD COLUMN "schedule_type" TEXT NOT NULL DEFAULT 'Open';

--bun:split

ALTER TABLE "stops" ADD COLUMN "count_late_override" INTEGER;

--bun:split

ALTER TABLE "stops" ADD COLUMN "count_detention_override" INTEGER;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "count_late_only_on_appointment_stops" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "count_detention_only_on_appointment_stops" INTEGER NOT NULL DEFAULT 0;
