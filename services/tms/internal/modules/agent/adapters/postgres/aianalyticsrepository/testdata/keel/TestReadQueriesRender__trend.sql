SELECT awu.model_key, $4::text, CASE WHEN awu.finished_at >= $5 THEN $6::bigint ELSE $7::bigint END, count(*), count(*) FILTER (WHERE awu.outcome = 'Completed'), coalesce(sum(awu.tool_calls), $8), coalesce(sum((awu.tool_failures_model + awu.tool_failures_tool)), $9), percentile_cont($10) WITHIN GROUP (ORDER BY awu.duration_ms ASC) FILTER (WHERE awu.outcome = 'Completed')
FROM ai_work_units awu
WHERE awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.task = 'AssistantChat' AND awu.finished_at >= $3 AND awu.finished_at >= $3
GROUP BY awu.model_key, $4::text, 3
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
-- $5 = 1790086400
-- $6 = 1790086400
-- $7 = 1790000000
-- $8 = 0
-- $9 = 0
-- $10 = 0.5
