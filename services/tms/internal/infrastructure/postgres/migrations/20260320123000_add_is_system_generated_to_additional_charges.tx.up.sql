ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "is_system_generated" boolean NOT NULL DEFAULT false;
