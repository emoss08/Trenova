ALTER TABLE "customers"
    DROP CONSTRAINT IF EXISTS "chk_customers_status_update_preference";

--bun:split

ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "status_update_recipients";

--bun:split

ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "status_update_preference";
