-- A finished restore may carry a note for staff (a restored audit trail that was marked, or left
-- broken), which the tenant agent reports as the restore's note; the reason stays for a failed
-- restore only, so the two are kept apart.
ALTER TABLE "cloud_tenants"
    ADD COLUMN IF NOT EXISTS "last_restore_note" varchar(500);

COMMENT ON COLUMN "cloud_tenants"."last_restore_note" IS 'One line for staff about a finished restore, such as an audit trail marked as not verified at its source; empty for a failed restore, whose words are in last_restore_reason';
