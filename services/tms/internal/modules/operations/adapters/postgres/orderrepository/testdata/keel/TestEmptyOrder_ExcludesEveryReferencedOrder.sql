SELECT o.id, o.business_unit_id, o.organization_id, o.customer_id, o.owner_id, o.entered_by_id, o.status, o.order_number, o.po_number, o.bol, o.currency_code, o.quoted_amount, o.base_amount, o.total_amount, o.version, o.created_at, o.updated_at
FROM orders o
WHERE o.business_unit_id = $1 AND o.organization_id = $2 AND o.id = $3 AND NOT EXISTS (
  SELECT 1
  FROM shipments s
  WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.order_id = o.id
) AND NOT EXISTS (
  SELECT 1
  FROM order_charges c
  WHERE c.business_unit_id = $1 AND c.organization_id = $2 AND c.order_id = o.id
) AND NOT EXISTS (
  SELECT 1
  FROM invoices i
  WHERE i.organization_id = $2 AND i.business_unit_id = $1 AND i.order_id = o.id
) AND NOT EXISTS (
  SELECT 1
  FROM billing_queue_items bqi
  WHERE bqi.organization_id = $2 AND bqi.business_unit_id = $1 AND bqi.order_id = o.id
)
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 'ord_01'
