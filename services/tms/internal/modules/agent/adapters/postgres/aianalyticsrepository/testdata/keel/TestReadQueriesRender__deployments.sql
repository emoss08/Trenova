SELECT awu.model_key, coalesce(awu.provider_id, $4), count(*), count(*) FILTER (WHERE awu.outcome = 'Completed'), count(*) FILTER (WHERE awu.outcome = 'HandedOff'), count(*) FILTER (WHERE awu.outcome = 'Stopped'), count(*) FILTER (WHERE awu.outcome = 'Failed'), coalesce(sum(awu.model_calls), $5), coalesce(sum(awu.failed_calls), $6), coalesce(sum(awu.tool_calls), $7), coalesce(sum(awu.tool_failures_model), $8), coalesce(sum(awu.tool_failures_tool), $9), CAST(coalesce(sum(awu.cost_usd), $10) AS text), count(awu.cost_usd), coalesce(sum(awu.input_tokens), $11), coalesce(sum(awu.output_tokens), $12), coalesce(sum(awu.cache_read_tokens), $13), coalesce(sum(awu.cache_write_tokens), $14), coalesce(sum(awu.advisor_calls), $15), coalesce(sum(awu.advisor_prompt_tokens), $16), coalesce(sum(awu.advisor_output_tokens), $17), CAST(coalesce(sum(awu.advisor_cost_usd), $18) AS text), coalesce(max(awu.finished_at), $19)
FROM ai_work_units awu
WHERE awu.business_unit_id = $1 AND awu.organization_id = $2 AND awu.task = 'AssistantChat' AND awu.finished_at >= $3
GROUP BY awu.model_key, 2
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = ''
-- $5 = 0
-- $6 = 0
-- $7 = 0
-- $8 = 0
-- $9 = 0
-- $10 = 0
-- $11 = 0
-- $12 = 0
-- $13 = 0
-- $14 = 0
-- $15 = 0
-- $16 = 0
-- $17 = 0
-- $18 = 0
-- $19 = 0
