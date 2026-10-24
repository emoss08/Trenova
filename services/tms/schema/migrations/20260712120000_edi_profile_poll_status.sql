ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_attempt_at" bigint;

ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_success_at" bigint;

ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_error" text;
