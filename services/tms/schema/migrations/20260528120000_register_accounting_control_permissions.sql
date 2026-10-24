WITH accounting_control_resources(resource, operations) AS (
    VALUES
        ('accounting_control', ARRAY['read', 'create', 'update', 'export', 'import']::text[]),
        ('billing_control', ARRAY['read', 'create', 'update', 'export', 'import']::text[]),
        ('invoice_adjustment_control', ARRAY['read', 'create', 'update', 'export', 'import']::text[]),
        ('account_type', ARRAY['read', 'create', 'update', 'export', 'import']::text[])
)
INSERT INTO resource_permissions(
    id,
    role_id,
    resource,
    operations,
    data_scope,
    created_at,
    updated_at
)
SELECT
    CONCAT('rp_', replace(gen_random_uuid()::text, '-', '')),
    r.id,
    acr.resource,
    acr.operations,
    'organization',
    EXTRACT(EPOCH FROM current_timestamp)::bigint,
    EXTRACT(EPOCH FROM current_timestamp)::bigint
FROM roles r
CROSS JOIN accounting_control_resources acr
WHERE r.is_system = true
  AND r.name = 'Organization Administrator'
  AND NOT EXISTS (
      SELECT 1
      FROM resource_permissions rp
      WHERE rp.role_id = r.id
        AND rp.resource = acr.resource
  );

