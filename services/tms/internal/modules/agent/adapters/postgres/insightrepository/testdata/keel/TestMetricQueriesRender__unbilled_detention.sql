SELECT l.id, l.name, count(*), CAST(coalesce(sum(do_.billable_amount), $7) AS text), CAST(coalesce(sum(do_.raw_dwell_minutes), $8) AS bigint)
FROM detention_occurrences do_
JOIN locations l ON l.business_unit_id = $1 AND l.organization_id = $2 AND l.id = do_.location_id
WHERE do_.business_unit_id = $1 AND do_.organization_id = $2 AND do_.clock_start_at >= $3 AND do_.clock_start_at < $4 AND do_.status IN ('Waived', 'NotBillable', 'Pending', 'Approved') AND do_.is_open = $5 AND do_.billable_amount > $6
GROUP BY l.id, l.name
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = 1792592000
-- $5 = false
-- $6 = 0
-- $7 = 0
-- $8 = 0
