DROP TABLE IF EXISTS "user_notification_preferences";

--bun:split
ALTER TABLE "users"
    DROP CONSTRAINT IF EXISTS "ck_users_away_delegate_not_self",
    DROP CONSTRAINT IF EXISTS "ck_users_availability_status",
    DROP CONSTRAINT IF EXISTS "ck_users_avatar_style",
    DROP COLUMN IF EXISTS "password_changed_at",
    DROP COLUMN IF EXISTS "away_note",
    DROP COLUMN IF EXISTS "away_delegate_user_id",
    DROP COLUMN IF EXISTS "away_back_on",
    DROP COLUMN IF EXISTS "focus_until",
    DROP COLUMN IF EXISTS "availability_status",
    DROP COLUMN IF EXISTS "avatar_style",
    DROP COLUMN IF EXISTS "phone_number",
    DROP COLUMN IF EXISTS "job_title";
