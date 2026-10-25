SELECT $4::text, coalesce(awu.provider_id, $5), awu.outcome, count(*), percentile_cont($6) WITHIN GROUP (ORDER BY awu.duration_ms ASC), percentile_cont($7) WITHIN GROUP (ORDER BY awu.duration_ms ASC), percentile_cont($8) WITHIN GROUP (ORDER BY awu.duration_ms ASC), percentile_cont($9) WITHIN GROUP (ORDER BY awu.duration_ms ASC)
FROM ai_work_units awu
WHERE awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.task = 'AssistantChat' AND awu.finished_at >= $3
GROUP BY $10, 2, awu.outcome
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
-- $5 = ''
-- $6 = 0.25
-- $7 = 0.5
-- $8 = 0.75
-- $9 = 0.95
-- $10 = ''
