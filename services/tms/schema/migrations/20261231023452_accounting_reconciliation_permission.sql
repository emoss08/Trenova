-- Where the accounting system and Trenova disagree, and the payments it recorded, are now their
-- own resource (accounting_reconciliation, on the Accounting add-on), apart from the outbound sync
-- queue (accounting_sync), which every plan's accounting integration includes. Every role and API
-- key that could read or act on reconciliation through accounting_sync keeps exactly that on
-- accounting_reconciliation, so nobody loses access on upgrade; whether the organization's plan
-- includes Accounting is still decided at request time. An administrator then takes
-- reconciliation away from people who should only work the sync queue.
INSERT INTO "resource_permissions"("id", "role_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
       rp."role_id",
       'accounting_reconciliation',
       ARRAY(
           SELECT op
           FROM unnest(ARRAY['read', 'update']::text[]) AS op
           WHERE op = ANY(rp."operations")
       ),
       rp."data_scope",
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "resource_permissions" rp
WHERE rp."resource" = 'accounting_sync'
  AND rp."operations" && ARRAY['read', 'update']::text[]
  AND NOT EXISTS (
      SELECT 1
      FROM "resource_permissions" existing
      WHERE existing."role_id" = rp."role_id"
        AND existing."resource" = 'accounting_reconciliation'
  );

INSERT INTO "api_key_permissions"("id", "api_key_id", "business_unit_id", "organization_id", "resource", "operations", "data_scope", "created_at", "updated_at")
SELECT CONCAT('akp_', replace(gen_random_uuid()::text, '-', '')),
       akp."api_key_id",
       akp."business_unit_id",
       akp."organization_id",
       'accounting_reconciliation',
       ARRAY(
           SELECT op
           FROM unnest(ARRAY['read', 'update']::text[]) AS op
           WHERE op = ANY(akp."operations")
       ),
       akp."data_scope",
       EXTRACT(EPOCH FROM current_timestamp)::bigint,
       EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM "api_key_permissions" akp
WHERE akp."resource" = 'accounting_sync'
  AND akp."operations" && ARRAY['read', 'update']::text[]
  AND NOT EXISTS (
      SELECT 1
      FROM "api_key_permissions" existing
      WHERE existing."api_key_id" = akp."api_key_id"
        AND existing."resource" = 'accounting_reconciliation'
  );

-- The watchtower items for differences and recorded payments move to their own source kind, read
-- under the new resource.
UPDATE "watchtower_items"
SET "source_kind" = 'AccountingReconciliation'
WHERE "source_kind" = 'AccountingSync'
  AND "subject_type" IN ('AccountingInboundChange', 'AccountingDriftFinding');
