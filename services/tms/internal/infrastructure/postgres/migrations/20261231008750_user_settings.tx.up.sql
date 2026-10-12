-- What a person says about themselves in Settings: a job title and phone their team can
-- see, the gradient their avatar falls back to, and whether they are available, focusing
-- or away (with who covers for them). The password's change time is kept so Settings can
-- say how old it is.
ALTER TABLE "users"
    ADD COLUMN IF NOT EXISTS "job_title" varchar(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "phone_number" varchar(30) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "avatar_style" varchar(20) NOT NULL DEFAULT 'auto',
    ADD COLUMN IF NOT EXISTS "availability_status" varchar(20) NOT NULL DEFAULT 'available',
    ADD COLUMN IF NOT EXISTS "focus_until" bigint,
    ADD COLUMN IF NOT EXISTS "away_back_on" bigint,
    ADD COLUMN IF NOT EXISTS "away_delegate_user_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "away_note" varchar(280) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "password_changed_at" bigint;

--bun:split
ALTER TABLE "users"
    ADD CONSTRAINT "ck_users_avatar_style" CHECK ("avatar_style" IN ('auto', 'dusk', 'lagoon', 'citrus', 'orchid', 'ember', 'mint', 'slate'));

--bun:split
ALTER TABLE "users"
    ADD CONSTRAINT "ck_users_availability_status" CHECK ("availability_status" IN ('available', 'focus', 'away'));

--bun:split
ALTER TABLE "users"
    ADD CONSTRAINT "ck_users_away_delegate_not_self" CHECK ("away_delegate_user_id" IS NULL OR "away_delegate_user_id" <> "id");

--bun:split
-- Where a person's notifications go outside the bell, and when they hold. The bell itself
-- always receives everything; these settings govern desktop and email delivery only.
CREATE TABLE IF NOT EXISTS "user_notification_preferences"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "disabled_event_types" text[] NOT NULL DEFAULT '{}',
    "desktop_mode" varchar(10) NOT NULL DEFAULT 'urgent',
    "email_mode" varchar(10) NOT NULL DEFAULT 'urgent',
    "email_digest" boolean NOT NULL DEFAULT TRUE,
    "sound" boolean NOT NULL DEFAULT TRUE,
    "quiet_hours_enabled" boolean NOT NULL DEFAULT FALSE,
    "quiet_hours_start" varchar(5) NOT NULL DEFAULT '19:00',
    "quiet_hours_end" varchar(5) NOT NULL DEFAULT '07:00',
    "quiet_hours_days" smallint[] NOT NULL DEFAULT '{1,2,3,4,5}',
    "quiet_hours_allow_urgent" boolean NOT NULL DEFAULT TRUE,
    "paused_until" bigint,
    "last_digest_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_user_notification_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_user_notification_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_notification_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_notification_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_user_notification_preferences_desktop_mode" CHECK ("desktop_mode" IN ('off', 'urgent', 'all')),
    CONSTRAINT "ck_user_notification_preferences_email_mode" CHECK ("email_mode" IN ('off', 'urgent', 'all')),
    CONSTRAINT "ck_user_notification_preferences_quiet_start" CHECK ("quiet_hours_start" ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT "ck_user_notification_preferences_quiet_end" CHECK ("quiet_hours_end" ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    CONSTRAINT "ck_user_notification_preferences_quiet_days" CHECK ("quiet_hours_days" <@ ARRAY[0, 1, 2, 3, 4, 5, 6]::smallint[])
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_user_notification_preferences_user" ON "user_notification_preferences"("organization_id", "business_unit_id", "user_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_notification_preferences_digest" ON "user_notification_preferences"("organization_id", "business_unit_id")
WHERE
    "email_mode" = 'all' AND "email_digest";

--bun:split
COMMENT ON TABLE "user_notification_preferences" IS 'Per-user delivery preferences for desktop and email notifications; the in-app bell is not governed by them';

--bun:split
SELECT
    trenova_rls.reconcile();
