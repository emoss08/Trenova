ALTER TABLE "organizations"
    DROP COLUMN IF EXISTS "asset_operations_enabled",
    DROP COLUMN IF EXISTS "brokerage_enabled";
