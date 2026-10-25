SELECT CAST(coalesce(sum(sm.distance) FILTER (WHERE (sm.created_at >= $6 AND sm.created_at <= $4)), $7) AS double precision), CAST(coalesce(sum(sm.distance) FILTER (WHERE ((sm.created_at >= $6 AND sm.created_at <= $4) AND sm.loaded = $8)), $9) AS double precision), CAST(coalesce(sum(sm.distance) FILTER (WHERE (sm.created_at >= $3 AND sm.created_at <= $10)), $11) AS double precision), CAST(coalesce(sum(sm.distance) FILTER (WHERE ((sm.created_at >= $3 AND sm.created_at <= $10) AND sm.loaded = $8)), $12) AS double precision)
FROM shipment_moves sm
WHERE sm.business_unit_id = $1 AND sm.organization_id = $2 AND (sm.created_at >= $3 AND sm.created_at <= $4) AND sm.distance IS NOT NULL AND sm.distance > $5
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 0
-- $4 = 1500
-- $5 = 0
-- $6 = 1000
-- $7 = 0
-- $8 = false
-- $9 = 0
-- $10 = 500
-- $11 = 0
-- $12 = 0
