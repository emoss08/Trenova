ALTER TABLE "edi_partners"
    ADD COLUMN IF NOT EXISTS "default_validation_profile_id" varchar(100);
