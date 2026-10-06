ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency";

--bun:split
ALTER TABLE "audit_entries"
    ADD CONSTRAINT "chk_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
        OR
        ("principal_type" = 'system' AND "user_id" IS NULL AND "api_key_id" IS NULL AND "principal_id" IS NOT NULL)
        OR
        ("principal_type" = 'agent' AND "api_key_id" IS NULL AND "principal_id" IS NOT NULL AND ("user_id" IS NULL OR "principal_id" <> "user_id"))
    );

COMMENT ON CONSTRAINT "chk_audit_entries_principal_consistency" ON "audit_entries" IS
    'An agent principal may name the system user an unattended run is attributed to; it is never also that user.';
