--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE IF EXISTS "timesheets"
    DROP CONSTRAINT IF EXISTS "fk_timesheets_payroll_export";

--bun:split
DROP TABLE IF EXISTS "time_clock_entries";

--bun:split
DROP TABLE IF EXISTS "payroll_exports";

--bun:split
DROP TABLE IF EXISTS "timesheets";

--bun:split
DROP TYPE IF EXISTS "payroll_export_status_enum";

--bun:split
DROP TYPE IF EXISTS "timesheet_status_enum";

--bun:split
DROP TYPE IF EXISTS "time_entry_source_enum";
