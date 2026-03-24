ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "fk_audit_entries_user";

ALTER TABLE "audit_entries"
    ALTER COLUMN "user_id" DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS "principal_type" varchar(50) NOT NULL DEFAULT 'session_user',
    ADD COLUMN IF NOT EXISTS "principal_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "api_key_id" varchar(100);

UPDATE "audit_entries"
SET
    "principal_type" = 'session_user',
    "principal_id" = "user_id"
WHERE "principal_id" IS NULL;

ALTER TABLE "audit_entries"
    ALTER COLUMN "principal_id" SET NOT NULL,
    ADD CONSTRAINT "fk_audit_entries_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_audit_entries_api_key" FOREIGN KEY ("api_key_id") REFERENCES "api_keys"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "chk_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key')),
    ADD CONSTRAINT "chk_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
    );

CREATE INDEX IF NOT EXISTS "idx_audit_entries_principal" ON "audit_entries"("principal_type", "principal_id");
CREATE INDEX IF NOT EXISTS "idx_audit_entries_api_key" ON "audit_entries"("api_key_id");
