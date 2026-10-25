SELECT $7::text, coalesce((
  SELECT sf_reviewed.id
  FROM edi_messages sf_reviewed
  WHERE sf_reviewed.business_unit_id = $1 AND sf_reviewed.organization_id = $2 AND sf_reviewed.transaction_set = '214' AND sf_reviewed.direction = 'Outbound' AND (sf_reviewed.payload_snapshot #>> $8) = $9 AND (sf_reviewed.payload_snapshot #>> $10) = $11 AND (sf_reviewed.payload_snapshot #>> $12) = $13
  ORDER BY sf_reviewed.generated_at DESC, sf_reviewed.id DESC LIMIT 1
), $14), coalesce((
  SELECT sf_resolved.id
  FROM edi_messages sf_resolved
  WHERE sf_resolved.business_unit_id = $1 AND sf_resolved.organization_id = $2 AND sf_resolved.transaction_set = '214' AND sf_resolved.direction = 'Outbound' AND (sf_resolved.payload_snapshot #>> $15) = $16 AND (sf_resolved.payload_snapshot #>> $17) = $18 AND (sf_resolved.payload_snapshot #>> $19) = $20
  ORDER BY sf_resolved.generated_at DESC, sf_resolved.id DESC LIMIT 1
), $21), em.id, coalesce(CAST(em.status AS text), $22), coalesce(CAST(em.delivery_status AS text), $23), coalesce(CAST(em.ack_status AS text), $24), coalesce((
  SELECT emve.message
  FROM edi_message_validation_errors emve
  WHERE emve.business_unit_id = $1 AND emve.organization_id = $2 AND emve.message_id = em.id AND emve.severity = 'Error'
  ORDER BY emve.created_at DESC, emve.id DESC LIMIT 1
), $25), coalesce(em.generated_at, $26)
FROM edi_messages em
WHERE em.business_unit_id = $1 AND em.organization_id = $2 AND em.transaction_set = '214' AND em.direction = 'Outbound' AND (em.payload_snapshot #>> $3) = $4 AND (em.payload_snapshot #>> $5) = $6
ORDER BY em.generated_at DESC, em.id DESC LIMIT 1
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = '{shipmentStatus,serviceFailureId}'
-- $4 = 'sf_1'
-- $5 = '{shipmentStatus,references,serviceFailureId}'
-- $6 = 'sf_1'
-- $7 = 'sf_1'
-- $8 = '{shipmentStatus,serviceFailureId}'
-- $9 = 'sf_1'
-- $10 = '{shipmentStatus,references,serviceFailureId}'
-- $11 = 'sf_1'
-- $12 = '{shipmentStatus,references,serviceFailure214Trigger}'
-- $13 = 'Reviewed'
-- $14 = ''
-- $15 = '{shipmentStatus,serviceFailureId}'
-- $16 = 'sf_1'
-- $17 = '{shipmentStatus,references,serviceFailureId}'
-- $18 = 'sf_1'
-- $19 = '{shipmentStatus,references,serviceFailure214Trigger}'
-- $20 = 'Resolved'
-- $21 = ''
-- $22 = ''
-- $23 = ''
-- $24 = ''
-- $25 = ''
-- $26 = 0
