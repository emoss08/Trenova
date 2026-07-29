--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE INDEX IF NOT EXISTS "idx_home_layout_presets_role_ids" ON "home_layout_presets" USING GIN("role_ids");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_home_layout_presets_tenant" ON "home_layout_presets"("organization_id", "business_unit_id");
