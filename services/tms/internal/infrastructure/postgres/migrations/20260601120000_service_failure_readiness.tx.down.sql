DELETE FROM resource_permissions rp
USING roles r
WHERE rp.role_id = r.id
  AND r.is_system = true
  AND r.name = 'Organization Administrator'
  AND rp.resource IN (
      'service_failure',
      'service_failure_reason_code'
  );

--bun:split
DELETE FROM "edi_source_context_schemas"
WHERE "id" = 'edisc_x12_214_out_shipment_status_v1';

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "shipment_comments"
        WHERE "user_id" IS NULL
    ) THEN
        RAISE EXCEPTION 'Cannot roll back nullable shipment comment users while system comments without users exist';
    END IF;
END $$;

ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "ck_shipment_comments_user_source",
    ALTER COLUMN "user_id" SET NOT NULL;

--bun:split
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM "audit_entries"
        WHERE "principal_type" = 'system'
    ) THEN
        RAISE EXCEPTION 'Cannot roll back system audit principal support while system audit entries exist';
    END IF;
END $$;

ALTER TABLE "audit_entries"
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency",
    DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_type";

ALTER TABLE "audit_entries"
    ADD CONSTRAINT "chk_audit_entries_principal_type" CHECK ("principal_type" IN ('session_user', 'api_key')),
    ADD CONSTRAINT "chk_audit_entries_principal_consistency" CHECK (
        ("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL AND "principal_id" = "user_id")
        OR
        ("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL AND "principal_id" = "api_key_id")
    );
