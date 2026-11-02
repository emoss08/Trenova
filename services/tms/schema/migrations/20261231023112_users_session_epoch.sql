-- A count of the times a person's credentials changed in a way that must end their sessions:
-- a password change or reset, or an administrator resetting their second factor. Each
-- session records the count it was issued under, and the periodic access check ends any
-- session whose count is behind, including one a sign-in finished after the change ended
-- the person's other sessions.
ALTER TABLE "users"
    ADD COLUMN IF NOT EXISTS "session_epoch" bigint NOT NULL DEFAULT 0;
