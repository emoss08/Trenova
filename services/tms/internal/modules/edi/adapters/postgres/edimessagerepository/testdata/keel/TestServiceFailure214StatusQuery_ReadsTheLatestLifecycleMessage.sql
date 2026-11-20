SELECT $5::text, coalesce((
  SELECT sf_reviewed.id
  FROM edi_messages sf_reviewed
  WHERE sf_reviewed.business_unit_id = $1 AND sf_reviewed.organization_id = $2 AND sf_reviewed.transaction_set = '214' AND sf_reviewed.direction = 'Outbound' AND (sf_reviewed.payload_snapshot #>> $6) = $7 AND (sf_reviewed.payload_snapshot #>> $8) = $9
  ORDER BY sf_reviewed.generated_at DESC, sf_reviewed.id DESC LIMIT 1
), $10), coalesce((
  SELECT sf_resolved.id
  FROM edi_messages sf_resolved
  WHERE sf_resolved.business_unit_id = $1 AND sf_resolved.organization_id = $2 AND sf_resolved.transaction_set = '214' AND sf_resolved.direction = 'Outbound' AND (sf_resolved.payload_snapshot #>> $11) = $12 AND (sf_resolved.payload_snapshot #>> $13) = $14
  ORDER BY sf_resolved.generated_at DESC, sf_resolved.id DESC LIMIT 1
), $15), em.id, coalesce(CAST(em.status AS text), $16), coalesce(CAST(em.delivery_status AS text), $17), coalesce(CAST(em.ack_status AS text), $18), coalesce((
  SELECT emve.message
  FROM edi_message_validation_errors emve
  WHERE emve.business_unit_id = $1 AND emve.organization_id = $2 AND emve.message_id = em.id AND emve.severity = 'Error'
  ORDER BY emve.created_at DESC, emve.id DESC LIMIT 1
), $19), coalesce(em.generated_at, $20)
FROM edi_messages em
WHERE em.business_unit_id = $1 AND em.organization_id = $2 AND em.transaction_set = '214' AND em.direction = 'Outbound' AND (em.payload_snapshot #>> $3) = $4
ORDER BY em.generated_at DESC, em.id DESC LIMIT 1
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = '{shipmentStatus,serviceFailureId}'
-- $4 = 'sf_1'
-- $5 = 'sf_1'
-- $6 = '{shipmentStatus,serviceFailureId}'
-- $7 = 'sf_1'
-- $8 = '{shipmentStatus,serviceFailureTrigger}'
-- $9 = 'Reviewed'
-- $10 = ''
-- $11 = '{shipmentStatus,serviceFailureId}'
-- $12 = 'sf_1'
-- $13 = '{shipmentStatus,serviceFailureTrigger}'
-- $14 = 'Resolved'
-- $15 = ''
-- $16 = ''
-- $17 = ''
-- $18 = ''
-- $19 = ''
-- $20 = 0
