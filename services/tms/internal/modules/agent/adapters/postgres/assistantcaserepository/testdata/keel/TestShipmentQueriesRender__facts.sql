SELECT s.status, s.actual_delivery_date, EXISTS (
  SELECT 1
  FROM documents d
  JOIN document_types dt ON dt.business_unit_id = $1 AND dt.organization_id = $2 AND dt.id = d.document_type_id
  WHERE d.organization_id = $2 AND d.business_unit_id = $1 AND d.resource_type = $4 AND d.resource_id = s.id AND d.is_current_version IS TRUE AND d.status NOT IN ('Rejected', 'Archived') AND upper(dt.code) = upper($5::text)
), ARRAY(
  SELECT DISTINCT d2.document_type_id
  FROM documents d2
  WHERE d2.organization_id = $2 AND d2.business_unit_id = $1 AND d2.resource_type = $4 AND d2.resource_id = s.id AND d2.is_current_version IS TRUE AND d2.status NOT IN ('Rejected', 'Archived') AND d2.document_type_id IS NOT NULL
), coalesce((
  SELECT jsonb_agg(jsonb_build_object($6::text, coalesce(c.name, $7), $8::text, (ca.status = 'Confirmed' OR EXISTS (
    SELECT 1
    FROM rate_confirmations rc
    WHERE rc.business_unit_id = $1 AND rc.organization_id = $2 AND rc.carrier_assignment_id = ca.id AND rc.status = 'Confirmed'
  ))) ORDER BY sm.sequence ASC)
  FROM carrier_assignments ca
  JOIN shipment_moves sm ON sm.business_unit_id = $1 AND sm.organization_id = $2 AND sm.id = ca.shipment_move_id
  LEFT JOIN carriers c ON c.business_unit_id = $1 AND c.organization_id = $2 AND c.id = ca.carrier_id
  WHERE ca.business_unit_id = $1 AND ca.organization_id = $2 AND sm.shipment_id = s.id AND ca.status <> 'Canceled'
), jsonb_build_array()), ARRAY(
  SELECT DISTINCT bqi.code
  FROM billing_queue_issues bqi
  JOIN billing_queue_items bqi2 ON bqi2.organization_id = $2 AND bqi2.business_unit_id = $1 AND bqi2.id = bqi.item_id
  WHERE bqi.business_unit_id = $1 AND bqi.organization_id = $2 AND bqi2.shipment_id = s.id AND bqi2.status <> 'Canceled' AND bqi.resolution_key IS NULL AND bqi.code IN ('AccessorialNotOnRateCon', 'ChargeOverRateCon')
), (
  SELECT count(*)
  FROM detention_occurrences do_
  WHERE do_.business_unit_id = $1 AND do_.organization_id = $2 AND do_.shipment_id = s.id AND do_.status IN ('Accruing')
), (
  SELECT count(*)
  FROM detention_occurrences do_2
  WHERE do_2.business_unit_id = $1 AND do_2.organization_id = $2 AND do_2.shipment_id = s.id AND do_2.status IN ('Pending', 'Disputed')
), (
  SELECT max(sc.created_at)
  FROM shipment_comments sc
  WHERE sc.business_unit_id = $1 AND sc.organization_id = $2 AND sc.shipment_id = s.id AND sc.type = 'CustomerUpdate' AND (sc.metadata #>> $9) = ANY($10)
), EXISTS (
  SELECT 1
  FROM billing_queue_items bqi3
  WHERE bqi3.organization_id = $2 AND bqi3.business_unit_id = $1 AND bqi3.shipment_id = s.id AND bqi3.status <> 'Canceled'
)
FROM shipments s
WHERE s.business_unit_id = $1 AND s.organization_id = $2 AND s.id = $3
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 'shp_1'
-- $4 = 'shipment'
-- $5 = 'POD'
-- $6 = 'carrierName'
-- $7 = ''
-- $8 = 'confirmed'
-- $9 = '{tool}'
-- $10 = '{customer_send_email,notify_shipment_delay}'
