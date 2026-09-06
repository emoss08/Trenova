--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
--
-- Time and attendance for staff paid by the hour.
--
-- The load-bearing decision is that a timesheet's totals are frozen when it is
-- submitted rather than recomputed on read. Everywhere else in this system a
-- roll-up is derived, because a stored one goes stale. Here the roll-up IS the
-- decision: it is what a manager approved and what payroll was paid from.
-- Recomputing it later would quietly change what somebody signed off, which is
-- the one thing a wage record must never do.
CREATE TYPE "time_entry_source_enum" AS ENUM(
    'Clock',
    'Portal',
    'Manual',
    'Import'
);

--bun:split
CREATE TYPE "timesheet_status_enum" AS ENUM(
    'Open',
    'Submitted',
    'Approved',
    'Rejected',
    'Locked'
);

--bun:split
CREATE TYPE "payroll_export_status_enum" AS ENUM(
    'Draft',
    'Generated',
    'Voided'
);

--bun:split
CREATE TABLE IF NOT EXISTS "timesheets"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "status" timesheet_status_enum NOT NULL DEFAULT 'Open',
    -- A period is one week, starting on the Sunday the rota starts on. One
    -- week start across the whole product, so a timesheet and a rota row line
    -- up without anybody converting between them.
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    -- The totals below are frozen at submit. While the sheet is Open they are
    -- recomputed from the entries on every write; once submitted they are the
    -- record of what was approved.
    "regular_minutes" integer NOT NULL DEFAULT 0,
    "overtime_minutes" integer NOT NULL DEFAULT 0,
    "paid_leave_minutes" integer NOT NULL DEFAULT 0,
    "entry_count" integer NOT NULL DEFAULT 0,
    -- Overtime is everything past this many minutes in the week. It is copied
    -- onto the sheet rather than read from a setting, so changing the rule
    -- next quarter cannot restate a week that was already approved.
    "overtime_threshold_minutes" integer NOT NULL DEFAULT 2400,
    "submitted_at" bigint,
    "submitted_by_id" varchar(100),
    "approved_at" bigint,
    "approved_by_id" varchar(100),
    "decision_note" varchar(500),
    "payroll_export_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_timesheets" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_timesheets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_timesheets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_timesheets_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_timesheets_period" CHECK ("period_end" > "period_start"),
    CONSTRAINT "chk_timesheets_minutes" CHECK ("regular_minutes" >= 0 AND "overtime_minutes" >= 0 AND "paid_leave_minutes" >= 0),
    CONSTRAINT "chk_timesheets_threshold" CHECK ("overtime_threshold_minutes" > 0)
);

--bun:split
-- One sheet per worker per week. A second one would split a week's hours
-- across two approvals and quietly halve somebody's overtime.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_timesheets_worker_period" ON "timesheets"("organization_id", "business_unit_id", "worker_id", "period_start");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_timesheets_queue" ON "timesheets"("organization_id", "business_unit_id", "status", "period_start" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "time_clock_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "timesheet_id" varchar(100),
    "source" time_entry_source_enum NOT NULL DEFAULT 'Clock',
    "clocked_in_at" bigint NOT NULL,
    -- Null means the worker is still on the clock. It is the only state a
    -- partial entry has, which is what makes "one open entry" enforceable.
    "clocked_out_at" bigint,
    -- Unpaid break taken inside the entry, subtracted from its length.
    "break_minutes" integer NOT NULL DEFAULT 0,
    "pay_code_id" varchar(100),
    "note" varchar(500),
    -- An edited entry says who changed it and why. A wage record altered by
    -- somebody with no reason recorded is not a record anybody can defend.
    "edited_by_id" varchar(100),
    "edit_reason" varchar(500),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_time_clock_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_time_clock_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_time_clock_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_time_clock_entries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_time_clock_entries_timesheet" FOREIGN KEY ("timesheet_id", "organization_id", "business_unit_id") REFERENCES "timesheets"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_time_clock_entries_pay_code" FOREIGN KEY ("pay_code_id", "organization_id", "business_unit_id") REFERENCES "pay_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_time_clock_entries_window" CHECK ("clocked_out_at" IS NULL OR "clocked_out_at" > "clocked_in_at"),
    CONSTRAINT "chk_time_clock_entries_break" CHECK ("break_minutes" >= 0)
);

--bun:split
-- A worker is on the clock once or not at all. Two open entries is somebody
-- being paid twice for the same hour.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_time_clock_entries_open" ON "time_clock_entries"("organization_id", "business_unit_id", "worker_id")
WHERE
    "clocked_out_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_time_clock_entries_worker_day" ON "time_clock_entries"("organization_id", "business_unit_id", "worker_id", "clocked_in_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_time_clock_entries_timesheet" ON "time_clock_entries"("organization_id", "business_unit_id", "timesheet_id");

--bun:split
CREATE TABLE IF NOT EXISTS "payroll_exports"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" payroll_export_status_enum NOT NULL DEFAULT 'Draft',
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "timesheet_count" integer NOT NULL DEFAULT 0,
    "regular_minutes" integer NOT NULL DEFAULT 0,
    "overtime_minutes" integer NOT NULL DEFAULT 0,
    "paid_leave_minutes" integer NOT NULL DEFAULT 0,
    "generated_at" bigint,
    "generated_by_id" varchar(100),
    "voided_at" bigint,
    "void_reason" varchar(500),
    "note" varchar(500),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_payroll_exports" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_payroll_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_payroll_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_payroll_exports_period" CHECK ("period_end" > "period_start")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_payroll_exports_period" ON "payroll_exports"("organization_id", "business_unit_id", "period_start" DESC);

--bun:split
ALTER TABLE "timesheets"
    ADD CONSTRAINT "fk_timesheets_payroll_export" FOREIGN KEY ("payroll_export_id", "organization_id", "business_unit_id") REFERENCES "payroll_exports"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;
