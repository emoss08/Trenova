SELECT wct.id, wct.name, bool_or(wct.is_required), count(DISTINCT wc.worker_id), min(wc.expires_at), count(DISTINCT wc.worker_id) FILTER (WHERE wc.expires_at < $5)
FROM worker_credentials wc
JOIN worker_credential_types wct ON wct.business_unit_id = $1 AND wct.organization_id = $2 AND wct.id = wc.credential_type_id
JOIN workers w ON w.business_unit_id = $1 AND w.organization_id = $2 AND w.id = wc.worker_id
WHERE wc.business_unit_id = $1 AND wc.organization_id = $2 AND wc.expires_at IS NOT NULL AND wc.expires_at > $3 AND wc.expires_at < $4 AND wc.archived_at IS NULL AND w.status = 'Active'
GROUP BY wct.id, wct.name
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1792592000
-- $5 = 1790000000
