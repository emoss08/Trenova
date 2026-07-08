ALTER TABLE "edi_communication_profiles"
    DROP COLUMN IF EXISTS "last_poll_attempt_at";

--bun:split
ALTER TABLE "edi_communication_profiles"
    DROP COLUMN IF EXISTS "last_poll_success_at";

--bun:split
ALTER TABLE "edi_communication_profiles"
    DROP COLUMN IF EXISTS "last_poll_error";
