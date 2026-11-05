-- A device acts for a person with no session of its own, so it keeps what the session that
-- approved it held: the roles active then, and the session epoch a credential change moves past.
ALTER TABLE "capture_pairings"
    ADD COLUMN IF NOT EXISTS "active_role_ids" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "session_epoch" bigint NOT NULL DEFAULT 0;

ALTER TABLE "capture_devices"
    ADD COLUMN IF NOT EXISTS "active_role_ids" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "session_epoch" bigint NOT NULL DEFAULT 0;
