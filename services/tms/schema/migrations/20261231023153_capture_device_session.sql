-- A device acts for a person with no session of its own, so it keeps what the session that
-- approved it held: the roles active then, and the session epoch a credential change moves past.
ALTER TABLE "capture_pairings"
    ADD COLUMN IF NOT EXISTS "active_role_ids" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "session_epoch" bigint NOT NULL DEFAULT 0;

ALTER TABLE "capture_devices"
    ADD COLUMN IF NOT EXISTS "active_role_ids" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "session_epoch" bigint NOT NULL DEFAULT 0;

-- A device paired before this change holds the epoch its person has now, so it is not revoked on
-- its next refresh for a credential change that happened before anyone recorded one.
UPDATE "capture_devices" AS d
SET "session_epoch" = u."session_epoch"
FROM "users" AS u
WHERE u."id" = d."user_id";
