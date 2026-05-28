DELETE FROM resource_permissions rp
USING roles r
WHERE rp.role_id = r.id
  AND r.is_system = true
  AND r.name = 'Organization Administrator'
  AND rp.resource IN (
      'accounting_control',
      'billing_control',
      'invoice_adjustment_control',
      'account_type'
  );

