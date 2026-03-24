DROP INDEX IF EXISTS "idx_audit_entries_api_key";
DROP INDEX IF EXISTS "idx_audit_entries_principal";

ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency",
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_type",
    DROP CONSTRAINT IF EXISTS "fk_audit_entries_api_key";

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "audit_entries"
        WHERE "principal_type" = 'api_key'
           OR "user_id" IS NULL
    ) THEN
        RAISE EXCEPTION 'Cannot roll back audit principal fields while API key audit entries exist';
    END IF;
END $$;

ALTER TABLE "audit_entries"
    DROP COLUMN IF EXISTS "api_key_id",
    DROP COLUMN IF EXISTS "principal_id",
    DROP COLUMN IF EXISTS "principal_type";

ALTER TABLE "audit_entries"
    ALTER COLUMN "user_id" SET NOT NULL;
