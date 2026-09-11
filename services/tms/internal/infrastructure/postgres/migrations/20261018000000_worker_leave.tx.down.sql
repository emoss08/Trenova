--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "worker_leave_entries";
--bun:split
DROP TABLE IF EXISTS "worker_leave_cases";
--bun:split
DROP TABLE IF EXISTS "leave_controls";
--bun:split
DROP TYPE IF EXISTS "leave_certification_status_enum";
--bun:split
DROP TYPE IF EXISTS "leave_frequency_enum";
--bun:split
DROP TYPE IF EXISTS "leave_case_status_enum";
--bun:split
DROP TYPE IF EXISTS "leave_measurement_method_enum";
