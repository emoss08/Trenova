ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_locked" boolean NOT NULL DEFAULT FALSE;
