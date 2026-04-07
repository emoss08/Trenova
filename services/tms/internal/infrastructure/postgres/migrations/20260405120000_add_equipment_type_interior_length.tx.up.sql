SET statement_timeout = 0;

ALTER TABLE "equipment_types"
    ADD COLUMN IF NOT EXISTS "interior_length" NUMERIC(10,2) DEFAULT NULL;
