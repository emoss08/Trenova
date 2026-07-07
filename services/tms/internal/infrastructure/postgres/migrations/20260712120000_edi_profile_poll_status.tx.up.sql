ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_attempt_at" bigint;

--bun:split
ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_success_at" bigint;

--bun:split
ALTER TABLE "edi_communication_profiles"
    ADD COLUMN IF NOT EXISTS "last_poll_error" text;
