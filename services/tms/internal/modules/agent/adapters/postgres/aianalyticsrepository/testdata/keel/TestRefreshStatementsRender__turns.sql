INSERT INTO ai_work_units AS awu (id, source, agent_definition_id, provider_id, model, model_key, task, outcome, decision_run_id, started_at_ms, finished_at, duration_ms, first_token_ms, model_calls, failed_calls, failovers, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd, advisor_calls, advisor_prompt_tokens, advisor_output_tokens, advisor_cost_usd, tool_calls, tool_failures_model, tool_failures_tool, refresh_stamp, business_unit_id, organization_id)
SELECT s.c1, s.c2, nullif(s.c3, ''), nullif(s.c4, ''), s.c5, s.c6, s.c7, s.c8, nullif(s.c9, ''), s.c10, s.c11, s.c12, s.c13, s.c14, s.c15, s.c16, s.c17, s.c18, s.c19, s.c20, s.c21, s.c22, s.c23, s.c24, s.c25, s.c26, s.c27, s.c28, s.c29, $1, $2
FROM (
  WITH owners (id, decision_run_id, provider_hint, model_hint, task_hint, status, at_limit, started_ms, finished_at) AS (
    SELECT at_.id, at_.run_id, nullif((at_.fingerprint #>> $30), $31), nullif((at_.fingerprint #>> $32), $33), $34::text, at_.status, coalesce(CAST((at_.fingerprint #>> $35) AS boolean), $36), (coalesce(nullif(at_.started_at, $37), at_.created_at) * $13::bigint), coalesce(at_.completed_at, at_.updated_at)
    FROM assistant_turns at_
    WHERE at_.business_unit_id = $1 AND at_.organization_id = $2 AND at_.updated_at >= $28 AND at_.updated_at < $29 AND at_.status IN ('Completed', 'Refused', 'Stopped', 'Failed') AND at_.origin <> 'Compaction'
  ), calls (owner_id, model_calls, failed_calls, failovers, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd, advisor_calls, advisor_prompt_tokens, advisor_output_tokens, advisor_cost_usd, first_start_ms, last_end_ms, first_token_ms, model, provider_id, task, agent_definition_id, answered, last_call_at) AS (
    SELECT aur.owner_id, count(*), count(*) FILTER (WHERE aur.succeeded = $38), count(*) FILTER (WHERE aur.failover IS TRUE), coalesce(sum(CASE WHEN aur.provider_kind = 'AnthropicMessages' THEN aur.input_tokens ELSE greatest(((aur.input_tokens - aur.cache_read_tokens) - aur.cache_write_tokens), 0) END), $39), coalesce(sum(aur.output_tokens), $40), coalesce(sum(aur.cache_read_tokens), $41), coalesce(sum(aur.cache_write_tokens), $42), sum(aur.cost_usd), coalesce(sum(aur.advisor_calls), $43), coalesce(sum(((aur.advisor_input_tokens + aur.advisor_cache_read_tokens) + aur.advisor_cache_write_tokens)), $44), coalesce(sum(aur.advisor_output_tokens), $45), sum(aur.advisor_cost_usd), min((coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) - aur.latency_ms)), max(coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint))), (array_agg(aur.first_token_ms ORDER BY (coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) - aur.latency_ms) ASC) FILTER (WHERE aur.first_token_ms IS NOT NULL))[1], (array_agg(aur.model ORDER BY aur.succeeded DESC, coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) DESC))[1], (array_agg(aur.provider_id ORDER BY aur.succeeded DESC, coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) DESC))[1], (array_agg(aur.task ORDER BY aur.succeeded DESC, coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) DESC))[1], (array_agg(aur.agent_definition_id ORDER BY coalesce(aur.recorded_at_ms, (aur.created_at * $13::bigint)) ASC) FILTER (WHERE aur.agent_definition_id IS NOT NULL))[1], bool_or(aur.succeeded), max(aur.created_at)
    FROM ai_usage_records aur
    WHERE aur.business_unit_id = $1 AND aur.organization_id = $2 AND aur.owner_kind = 'AssistantTurn' AND aur.owner_id IN (SELECT o2.id FROM owners o2)
    GROUP BY aur.owner_id
  ), steps (owner_id, tool_calls, tool_failures_model, tool_failures_tool, last_end_ms) AS (
    SELECT ars.owner_id, count(*), count(*) FILTER (WHERE c2.cause = ANY($49)), count(*) FILTER (WHERE (c2.cause <> $50 AND c2.cause <> ALL($51))), max(coalesce(ars.settled_at_ms, coalesce(ars.started_at_ms, (ars.updated_at * $13::bigint))))
    FROM agent_run_steps ars
    CROSS JOIN LATERAL (
      SELECT CASE WHEN ars.status <> $52 THEN $4::text WHEN coalesce((ars.outcome #>> $53), $54) = ANY($55) THEN (ars.outcome #>> $53) WHEN ((ars.outcome #>> $56) = $57 AND (ars.outcome #>> $58) = $59) THEN $60::text WHEN (ars.outcome #>> $56) = $61 THEN $62::text WHEN (ars.outcome #>> $56) = $63 THEN $64::text WHEN ((ars.outcome #>> $56) = $65 AND (ars.outcome #>> $58) = $66) THEN $60::text WHEN (ars.outcome #>> $56) = ANY($67) THEN $68::text WHEN (lower(coalesce((ars.outcome #>> $69), $70)) LIKE $71 OR lower(coalesce((ars.outcome #>> $69), $70)) LIKE $72 OR lower(coalesce((ars.outcome #>> $69), $70)) LIKE $73) THEN $74::text ELSE $75::text END AS cause
    ) c2
    WHERE ars.business_unit_id = $1 AND ars.organization_id = $2 AND ars.owner_kind = $46 AND ars.owner_id IN (SELECT o3.id FROM owners o3) AND ars.kind = $47 AND ars.status <> $48
    GROUP BY ars.owner_id
  ), handed (run_id) AS (
    SELECT DISTINCT ae.run_id
    FROM agent_exceptions ae
    WHERE ae.business_unit_id = $1 AND ae.organization_id = $2 AND ae.run_id IN (
      SELECT o4.decision_run_id
      FROM owners o4
      WHERE o4.decision_run_id IS NOT NULL
    )
  )
  SELECT o.id, $3::text, c.agent_definition_id, coalesce(c.provider_id, o.provider_hint), coalesce(c.model, o.model_hint), $4::text, coalesce(c.task, o.task_hint), CASE WHEN o.status = $5 THEN $6::text WHEN o.status = $7 THEN $8::text WHEN o.at_limit IS TRUE THEN $8::text WHEN h.run_id IS NOT NULL THEN $9::text ELSE $10::text END, o.decision_run_id, coalesce(c.first_start_ms, o.started_ms), o.finished_at, greatest(0, (greatest(coalesce(c.last_end_ms, $11), coalesce(s.last_end_ms, $12), CASE WHEN (c.last_end_ms IS NULL AND s.last_end_ms IS NULL) THEN (o.finished_at * $13::bigint) ELSE 0 END) - coalesce(c.first_start_ms, o.started_ms))), c.first_token_ms, coalesce(c.model_calls, $14), coalesce(c.failed_calls, $15), coalesce(c.failovers, $16), coalesce(c.input_tokens, $17), coalesce(c.output_tokens, $18), coalesce(c.cache_read_tokens, $19), coalesce(c.cache_write_tokens, $20), c.cost_usd, coalesce(c.advisor_calls, $21), coalesce(c.advisor_prompt_tokens, $22), coalesce(c.advisor_output_tokens, $23), c.advisor_cost_usd, coalesce(s.tool_calls, $24), coalesce(s.tool_failures_model, $25), coalesce(s.tool_failures_tool, $26), $27::bigint
  FROM owners o
  LEFT JOIN calls c ON c.owner_id = o.id
  LEFT JOIN steps s ON s.owner_id = o.id
  LEFT JOIN handed h ON h.run_id = o.decision_run_id
  WHERE coalesce(c.model, o.model_hint) IS NOT NULL
) AS s(c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12, c13, c14, c15, c16, c17, c18, c19, c20, c21, c22, c23, c24, c25, c26, c27, c28, c29)
ON CONFLICT (id, business_unit_id, organization_id) DO UPDATE SET source = EXCLUDED.source, agent_definition_id = EXCLUDED.agent_definition_id, provider_id = EXCLUDED.provider_id, model = EXCLUDED.model, task = EXCLUDED.task, outcome = EXCLUDED.outcome, decision_run_id = EXCLUDED.decision_run_id, started_at_ms = EXCLUDED.started_at_ms, finished_at = EXCLUDED.finished_at, duration_ms = EXCLUDED.duration_ms, first_token_ms = EXCLUDED.first_token_ms, model_calls = EXCLUDED.model_calls, failed_calls = EXCLUDED.failed_calls, failovers = EXCLUDED.failovers, input_tokens = EXCLUDED.input_tokens, output_tokens = EXCLUDED.output_tokens, cache_read_tokens = EXCLUDED.cache_read_tokens, cache_write_tokens = EXCLUDED.cache_write_tokens, cost_usd = EXCLUDED.cost_usd, advisor_calls = EXCLUDED.advisor_calls, advisor_prompt_tokens = EXCLUDED.advisor_prompt_tokens, advisor_output_tokens = EXCLUDED.advisor_output_tokens, advisor_cost_usd = EXCLUDED.advisor_cost_usd, tool_calls = EXCLUDED.tool_calls, tool_failures_model = EXCLUDED.tool_failures_model, tool_failures_tool = EXCLUDED.tool_failures_tool, refresh_stamp = EXCLUDED.refresh_stamp, model_key = CASE WHEN awu.model = EXCLUDED.model THEN awu.model_key ELSE $4::text END
WHERE (ROW(awu.source, awu.agent_definition_id, awu.provider_id, awu.model, awu.task, awu.outcome, awu.decision_run_id, awu.started_at_ms, awu.finished_at, awu.duration_ms, awu.first_token_ms, awu.model_calls, awu.failed_calls, awu.failovers, awu.input_tokens, awu.output_tokens, awu.cache_read_tokens, awu.cache_write_tokens, awu.cost_usd, awu.advisor_calls, awu.advisor_prompt_tokens, awu.advisor_output_tokens, awu.advisor_cost_usd, awu.tool_calls, awu.tool_failures_model, awu.tool_failures_tool) IS DISTINCT FROM ROW(EXCLUDED.source, EXCLUDED.agent_definition_id, EXCLUDED.provider_id, EXCLUDED.model, EXCLUDED.task, EXCLUDED.outcome, EXCLUDED.decision_run_id, EXCLUDED.started_at_ms, EXCLUDED.finished_at, EXCLUDED.duration_ms, EXCLUDED.first_token_ms, EXCLUDED.model_calls, EXCLUDED.failed_calls, EXCLUDED.failovers, EXCLUDED.input_tokens, EXCLUDED.output_tokens, EXCLUDED.cache_read_tokens, EXCLUDED.cache_write_tokens, EXCLUDED.cost_usd, EXCLUDED.advisor_calls, EXCLUDED.advisor_prompt_tokens, EXCLUDED.advisor_output_tokens, EXCLUDED.advisor_cost_usd, EXCLUDED.tool_calls, EXCLUDED.tool_failures_model, EXCLUDED.tool_failures_tool))
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 'AssistantTurn'
-- $4 = ''
-- $5 = 'Failed'
-- $6 = 'Failed'
-- $7 = 'Stopped'
-- $8 = 'Stopped'
-- $9 = 'HandedOff'
-- $10 = 'Completed'
-- $11 = 0
-- $12 = 0
-- $13 = 1000
-- $14 = 0
-- $15 = 0
-- $16 = 0
-- $17 = 0
-- $18 = 0
-- $19 = 0
-- $20 = 0
-- $21 = 0
-- $22 = 0
-- $23 = 0
-- $24 = 0
-- $25 = 0
-- $26 = 0
-- $27 = 7
-- $28 = 1790000000
-- $29 = 1790003600
-- $30 = '{providerId}'
-- $31 = ''
-- $32 = '{model}'
-- $33 = ''
-- $34 = 'AssistantChat'
-- $35 = '{stoppedAtLimit}'
-- $36 = false
-- $37 = 0
-- $38 = false
-- $39 = 0
-- $40 = 0
-- $41 = 0
-- $42 = 0
-- $43 = 0
-- $44 = 0
-- $45 = 0
-- $46 = 'AssistantTurn'
-- $47 = 'Tool'
-- $48 = 'Started'
-- $49 = '{WrongArguments,RepeatedCall,ToolNotHeld}'
-- $50 = ''
-- $51 = '{WrongArguments,RepeatedCall,ToolNotHeld}'
-- $52 = 'Failed'
-- $53 = '{cause}'
-- $54 = ''
-- $55 = '{WrongArguments,ToolTimedOut,RepeatedCall,BlockedByRule,ToolError,ToolNotHeld}'
-- $56 = '{verdict}'
-- $57 = 'invalid'
-- $58 = '{reason}'
-- $59 = 'no such tool is registered'
-- $60 = 'ToolNotHeld'
-- $61 = 'invalid'
-- $62 = 'WrongArguments'
-- $63 = 'duplicate'
-- $64 = 'RepeatedCall'
-- $65 = 'denied'
-- $66 = 'the agent does not hold the tool'
-- $67 = '{denied,over_budget}'
-- $68 = 'BlockedByRule'
-- $69 = '{reason}'
-- $70 = ''
-- $71 = '%deadline exceeded%'
-- $72 = '%timed out%'
-- $73 = '%timeout%'
-- $74 = 'ToolTimedOut'
-- $75 = 'ToolError'
