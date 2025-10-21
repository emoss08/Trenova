--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TYPE IF EXISTS "business_unit_status" CASCADE;

DROP TABLE IF EXISTS "business_units" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_business_units_name" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_business_units_created_at" CASCADE;

