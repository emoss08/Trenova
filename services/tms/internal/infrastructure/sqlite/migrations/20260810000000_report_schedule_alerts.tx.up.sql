-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260810000000_report_schedule_alerts.tx.up.sql

ALTER TABLE "report_schedules" ADD COLUMN "alert" TEXT;

--bun:split

ALTER TABLE "report_schedules" ADD COLUMN "alert_firing" INTEGER NOT NULL DEFAULT 0;
