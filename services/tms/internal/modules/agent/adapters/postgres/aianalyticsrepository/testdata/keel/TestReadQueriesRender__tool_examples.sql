WITH candidates (unit_id, finished_at) AS (
  SELECT awut.unit_id, max(awut.finished_at)
  FROM ai_work_unit_tools awut
  WHERE awut.organization_id = $12 AND awut.business_unit_id = $11 AND awut.finished_at >= $13 AND awut.task = 'AssistantChat' AND awut.tool_name = $14 AND awut.provider_id = $15 AND awut.cause <> $16
  GROUP BY awut.unit_id
  ORDER BY max(awut.finished_at) DESC LIMIT 125
)
SELECT awu.id, awu.source, ars.tool_name, left(coalesce(nullif((ars.outcome #>> $4), $5), coalesce((ars.outcome #>> $6), $7)), $8), coalesce(awu.agent_definition_id, $9), coalesce((ars.settled_at_ms / $10::bigint), ars.updated_at)
FROM candidates c
JOIN ai_work_units awu ON awu.business_unit_id = $11 AND awu.organization_id = $12 AND awu.id = c.unit_id
JOIN agent_run_steps ars ON ars.business_unit_id = $11 AND ars.organization_id = $12 AND ars.owner_id = c.unit_id
WHERE ars.kind = $1 AND ars.status = $2 AND ars.tool_name = $3
ORDER BY ars.updated_at DESC, ars.id DESC LIMIT 5
-- $1 = 'Tool'
-- $2 = 'Failed'
-- $3 = 'search'
-- $4 = '{content}'
-- $5 = ''
-- $6 = '{reason}'
-- $7 = ''
-- $8 = 300
-- $9 = ''
-- $10 = 1000
-- $11 = 'bu_1'
-- $12 = 'org_1'
-- $13 = 1790000000
-- $14 = 'search'
-- $15 = 'aip_1'
-- $16 = ''
