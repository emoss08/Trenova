DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "access_policies"
        GROUP BY "organization_id", "business_unit_id", lower("name")
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'access_policies contains duplicate rows for organization_id/business_unit_id/name; deduplicate before applying access policy uniqueness';
    END IF;
END $$;

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "access_policies"
        WHERE "enabled" = TRUE
        GROUP BY "organization_id", "business_unit_id", "resource", "operation", "conditions"
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'access_policies contains duplicate enabled decision scopes; deduplicate before applying access policy uniqueness';
    END IF;
END $$;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_access_policies_name_tenant"
    ON "access_policies"(lower("name"), "organization_id", "business_unit_id");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_access_policies_enabled_scope_tenant"
    ON "access_policies"("organization_id", "business_unit_id", "resource", "operation", "conditions")
    WHERE "enabled" = TRUE;
