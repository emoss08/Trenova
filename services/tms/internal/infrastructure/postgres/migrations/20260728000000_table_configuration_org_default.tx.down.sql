DROP INDEX IF EXISTS "idx_table_configurations_org_default";

--bun:split

ALTER TABLE "table_configurations"
    DROP COLUMN IF EXISTS "is_org_default";
