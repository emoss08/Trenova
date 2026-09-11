--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "ifta_tax_rates";
--bun:split
DROP TABLE IF EXISTS "ifta_jurisdictions";
--bun:split
DROP TYPE IF EXISTS "ifta_jurisdiction_status_enum";
--bun:split
DROP TYPE IF EXISTS "ifta_fuel_type_enum";
