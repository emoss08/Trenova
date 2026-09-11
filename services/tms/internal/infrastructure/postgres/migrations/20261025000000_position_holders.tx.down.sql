--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_user_organization_memberships_position";

--bun:split
ALTER TABLE IF EXISTS "user_organization_memberships"
    DROP CONSTRAINT IF EXISTS "fk_user_organization_memberships_position";

--bun:split
ALTER TABLE IF EXISTS "user_organization_memberships"
    DROP COLUMN IF EXISTS "position_id";
