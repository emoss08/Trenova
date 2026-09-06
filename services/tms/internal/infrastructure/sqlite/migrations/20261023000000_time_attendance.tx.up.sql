-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261023000000_time_attendance.tx.up.sql

CREATE TABLE IF NOT EXISTS "timesheets"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "regular_minutes" INTEGER NOT NULL DEFAULT 0,
    "overtime_minutes" INTEGER NOT NULL DEFAULT 0,
    "paid_leave_minutes" INTEGER NOT NULL DEFAULT 0,
    "entry_count" INTEGER NOT NULL DEFAULT 0,
    "overtime_threshold_minutes" INTEGER NOT NULL DEFAULT 2400,
    "submitted_at" INTEGER,
    "submitted_by_id" TEXT,
    "approved_at" INTEGER,
    "approved_by_id" TEXT,
    "decision_note" TEXT,
    "payroll_export_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_timesheets" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_timesheets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_timesheets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_timesheets_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_timesheets_period" CHECK ("period_end" > "period_start"),
    CONSTRAINT "chk_timesheets_minutes" CHECK ("regular_minutes" >= 0 AND "overtime_minutes" >= 0 AND "paid_leave_minutes" >= 0),
    CONSTRAINT "chk_timesheets_threshold" CHECK ("overtime_threshold_minutes" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_timesheets_worker_period" ON "timesheets" ("organization_id", "business_unit_id", "worker_id", "period_start");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_timesheets_queue" ON "timesheets" ("organization_id", "business_unit_id", "status", "period_start" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "time_clock_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "timesheet_id" TEXT,
    "source" TEXT NOT NULL DEFAULT 'Clock',
    "clocked_in_at" INTEGER NOT NULL,
    "clocked_out_at" INTEGER,
    "break_minutes" INTEGER NOT NULL DEFAULT 0,
    "pay_code_id" TEXT,
    "note" TEXT,
    "edited_by_id" TEXT,
    "edit_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_time_clock_entries_open" ON "time_clock_entries" ("organization_id", "business_unit_id", "worker_id")WHERE
    "clocked_out_at" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_time_clock_entries_worker_day" ON "time_clock_entries" ("organization_id", "business_unit_id", "worker_id", "clocked_in_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_time_clock_entries_timesheet" ON "time_clock_entries" ("organization_id", "business_unit_id", "timesheet_id");

--bun:split

CREATE TABLE IF NOT EXISTS "payroll_exports"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "timesheet_count" INTEGER NOT NULL DEFAULT 0,
    "regular_minutes" INTEGER NOT NULL DEFAULT 0,
    "overtime_minutes" INTEGER NOT NULL DEFAULT 0,
    "paid_leave_minutes" INTEGER NOT NULL DEFAULT 0,
    "generated_at" INTEGER,
    "generated_by_id" TEXT,
    "voided_at" INTEGER,
    "void_reason" TEXT,
    "note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_payroll_exports" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_payroll_exports_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_payroll_exports_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_payroll_exports_period" CHECK ("period_end" > "period_start")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_payroll_exports_period" ON "payroll_exports" ("organization_id", "business_unit_id", "period_start" DESC);
