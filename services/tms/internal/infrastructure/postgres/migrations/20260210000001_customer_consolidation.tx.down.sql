ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "consolidation_priority";

--bun:split
ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "exclusive_consolidation";

--bun:split
ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "allow_consolidation";
