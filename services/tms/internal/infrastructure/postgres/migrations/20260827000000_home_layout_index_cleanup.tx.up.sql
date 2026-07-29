--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Role matching happens in Go after the tenant's presets are loaded, so no
-- query ever filters on role_ids and the GIN index only taxes writes.
DROP INDEX IF EXISTS "idx_home_layout_presets_role_ids";

--bun:split
-- (organization_id, business_unit_id) is a prefix of
-- idx_home_layout_presets_priority, which serves the same scans.
DROP INDEX IF EXISTS "idx_home_layout_presets_tenant";
