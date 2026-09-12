--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "shift_swap_requests";
--bun:split
DROP TABLE IF EXISTS "worker_availability_preferences";
--bun:split
DROP TABLE IF EXISTS "worker_shift_assignments";
--bun:split
DROP TABLE IF EXISTS "shift_templates";
--bun:split
DROP TYPE IF EXISTS "shift_swap_status_enum";
--bun:split
DROP TYPE IF EXISTS "availability_preference_enum";
