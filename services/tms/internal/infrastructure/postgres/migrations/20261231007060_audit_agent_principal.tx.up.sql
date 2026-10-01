ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency",
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_type";

--bun:split
ALTER TABLE "audit_entries"
    ADD CONSTRAINT "chk_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key', 'system', 'agent')),
    ADD CONSTRAINT "chk_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
        OR
        ("principal_type" IN ('system', 'agent') AND "user_id" IS NULL AND "api_key_id" IS NULL AND "principal_id" IS NOT NULL)
    );
