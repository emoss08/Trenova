ALTER TABLE "table_configurations"
    ADD COLUMN IF NOT EXISTS "is_org_default" boolean NOT NULL DEFAULT FALSE;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_table_configurations_org_default"
    ON "table_configurations" ("organization_id", "business_unit_id", "resource")
    WHERE "is_org_default" = TRUE;
