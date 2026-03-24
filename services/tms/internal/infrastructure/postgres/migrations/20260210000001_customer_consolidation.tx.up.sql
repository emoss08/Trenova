ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "allow_consolidation" boolean NOT NULL DEFAULT TRUE;

--bun:split
ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "exclusive_consolidation" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "consolidation_priority" integer NOT NULL DEFAULT 1;
